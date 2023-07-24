// Copyright 2021-2023
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/stac"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// ModifyCollection creates a new collection in the database
// POST /collections/:collectionId/items
func CreateItems(c *fiber.Ctx) error {
	// validate passed JSON
	itemsRaw := c.Body()
	items := make(map[string]*json.RawMessage)

	if err := json.Unmarshal(itemsRaw, &items); err != nil {
		log.Error().Err(err).Str("RequestBody", string(itemsRaw)).Msg("cannot unmarshal body to items")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "JSON parse failed; items must be a valid JSON object",
		})
	}

	// switch based on geojson type Feature or FeatureCollection
	var geojsonType *json.RawMessage
	var ok bool
	if geojsonType, ok = items["type"]; !ok {
		log.Error().Str("RequestBody", string(itemsRaw)).Msg("items missing type field - must be a valid geojson object")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "items missing type field - must be a valid geojson object",
		})
	}

	geojsonTypeStr := string(*geojsonType)
	switch geojsonTypeStr {
	case "Feature":
		return createFeature(c, items, itemsRaw)
	case "FeatureCollection":
		return createFeatureCollection(c, items, itemsRaw)
	default:
		log.Error().Str("type", geojsonTypeStr).Msg("invalid geojson type")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "invalid geojson type - must be one of 'Feature' or 'FeatureCollection'",
		})
	}
}

func createFeature(c *fiber.Ctx, items map[string]*json.RawMessage, itemsRaw []byte) error {
	ctx := context.Background()
	collectionId := c.Params("collectionId")
	var err error

	// validate collection matches the expected collection
	if err = stac.ValidateCollectionIDsMatch(c, items, collectionId); err != nil {
		return err
	}

	// validate item itemId
	var itemId string
	if itemId, err = stac.ValidateID(c, items); err != nil {
		return err
	}

	itemsJson, err := json.Marshal(items)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal items to JSON")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "failed to marshal JSON for items",
		})
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT create_item($1::text::jsonb)", itemsJson); err != nil {
		log.Error().Err(err).Str("id", itemId).Str("raw", string(itemsRaw)).Msg("failed to create item")
		c.Status(fiber.ErrConflict.Code)
		return c.JSON(stac.Error{
			Code:        "CreateItemFailed",
			Description: "failed to create item",
		})
	}

	return itemFromID(c, collectionId, itemId)
}

func createFeatureCollection(c *fiber.Ctx, items map[string]*json.RawMessage, itemsRaw []byte) error {
	ctx := context.Background()
	collectionId := c.Params("collectionId")

	// for each feature validate collection matches the expected collection
	var featuresRaw *json.RawMessage
	var ok bool
	if featuresRaw, ok = items["features"]; !ok {
		log.Error().Str("raw", string(itemsRaw)).Msg("failed to get features - object invalid")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "objects of type 'FeatureCollection' must have a 'features' field",
		})
	}

	var features []map[string]*json.RawMessage
	if err := json.Unmarshal(*featuresRaw, &features); err != nil {
		log.Error().Err(err).Str("raw", string(itemsRaw)).Msg("unmarshal geojson features failed")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "objects of type 'FeatureCollection' must have a valid 'features' field",
		})
	}

	itemIds := make([]string, len(features))
	for idx, feature := range features {
		if err := stac.ValidateCollectionIDsMatch(c, feature, collectionId); err != nil {
			log.Error().Int("FeatureIndex", idx).Msg("failed collection ID match validation")
			return err
		}
		itemId, err := stac.ValidateID(c, feature)
		if err != nil {
			log.Error().Int("FeatureIndex", idx).Msg("failed ID validation")
			return err
		}
		itemIds[idx] = itemId
	}

	// validation has passed, create items
	itemsJson, err := json.Marshal(items)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal items to JSON")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: "failed to marshal JSON for items",
		})
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT create_items($1::text::jsonb)", itemsJson); err != nil {
		log.Error().Err(err).Strs("id", itemIds).Str("raw", string(itemsRaw)).Msg("failed to create item")
		c.Status(fiber.ErrConflict.Code)
		return c.JSON(stac.Error{
			Code:        "CreateItemFailed",
			Description: "failed to create item",
		})
	}

	return itemFromIDs(c, itemIds)
}

// Item returns details of a specific item
// GET /collections/:collectionId/items/:itemId
func Item(c *fiber.Ctx) error {
	collectionId := c.Params("collectionId")
	itemId := c.Params("itemId")

	return itemFromID(c, collectionId, itemId)
}

func itemFromID(c *fiber.Ctx, collectionId string, itemId string) error {
	ctx := context.Background()
	baseUrl := getBaseUrl(c)

	pool := database.GetInstance(ctx)

	// make sure the requested collection exists
	row := pool.QueryRow(ctx, "SELECT id FROM pgstac.collections WHERE id=$1", collectionId)
	var dbResult string
	if err := row.Scan(&dbResult); err != nil {
		log.Error().Err(err).Str("collectionId", collectionId).Msg("collection does not exist in database")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Error{
			Code:        stac.NotFoundError,
			Description: "could not query collections table",
		})
	}

	// create CQL search criteria
	conf := json.RawMessage(`{"nohydrate": false}`)
	cql := stac.CQL{
		Collections: []string{collectionId},
		Ids:         []string{itemId},
		Conf:        &conf,
		Limit:       1,
	}

	// do the search
	featureCollection, err := stac.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        stac.ServerError,
			Description: "stac search returned an error",
		})
	}

	// enrich links
	var myItem map[string]*json.RawMessage
	for _, item := range featureCollection.Features {
		var myLinksJson json.RawMessage
		var itemId string
		var links []stac.Link

		if err := json.Unmarshal(*item["id"], &itemId); err != nil {
			log.Error().Err(err).Msg("error de-serializing id")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error de-serializing item id",
			})
		}

		if err := json.Unmarshal(*item["links"], &links); err != nil {
			log.Error().Err(err).Msg("error de-serializing link")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error de-serializing item link",
			})
		}
		for idx, link := range links {
			if link.Rel == "collection" {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseUrl, collectionId)
			}
			links[idx] = link
		}

		links = stac.AddLink(links, baseUrl, "parent", fmt.Sprintf("/collections/%s", collectionId), "application/json")
		links = stac.AddLink(links, baseUrl, "root", "/", "application/json")
		links = stac.AddLink(links, baseUrl, "self", fmt.Sprintf("/collections/%s/items/%s", collectionId, itemId), "application/geo+json")

		myLinksJson, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("error serializing links")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error serializing item links",
			})
		}

		item["links"] = &myLinksJson
		myItem = item
	}

	return c.JSON(myItem)
}

// Items returns a list of items in a collection
// GET /collections/:collectionId/items
func Items(c *fiber.Ctx) error {
	ctx := context.Background()
	baseUrl := getBaseUrl(c)
	collectionId := c.Params("collectionId")

	pool := database.GetInstance(ctx)

	// make sure the requested collection exists
	row := pool.QueryRow(ctx, "SELECT id FROM pgstac.collections WHERE id=$1", collectionId)
	var dbResult string
	if err := row.Scan(&dbResult); err != nil {
		log.Error().Err(err).Str("collectionId", collectionId).Msg("collection does not exist in database")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Error{
			Code:        stac.NotFoundError,
			Description: "could not query collections table",
		})
	}

	// do the search
	cql, err := getCQLFromQuery(c)
	if err != nil {
		// http response and logging handled by getCQLFromQuery
		return err
	}
	cql.Collections = []string{collectionId}
	featureCollection, err := stac.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        stac.ServerError,
			Description: "stac search returned an error",
		})
	}

	// enrich links
	for _, item := range featureCollection.Features {
		var myLinksJson json.RawMessage
		var itemId string
		var links []stac.Link

		if err := json.Unmarshal(*item["id"], &itemId); err != nil {
			log.Error().Err(err).Msg("error de-serializing id")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error de-serializing item id",
			})
		}

		if err := json.Unmarshal(*item["links"], &links); err != nil {
			log.Error().Err(err).Msg("error de-serializing link")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error de-serializing item link",
			})
		}
		for idx, link := range links {
			if link.Rel == "collection" {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseUrl, collectionId)
			}
			links[idx] = link
		}

		links = stac.AddLink(links, baseUrl, "parent", fmt.Sprintf("/collections/%s", collectionId), "application/json")
		links = stac.AddLink(links, baseUrl, "root", "/", "application/json")
		links = stac.AddLink(links, baseUrl, "self", fmt.Sprintf("/collections/%s/items/%s", collectionId, itemId), "application/geo+json")

		myLinksJson, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("error serializing links")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error serializing item links",
			})
		}

		item["links"] = &myLinksJson
	}

	// overall links
	overallLinks := make([]stac.Link, 0, 4)
	overallLinks = stac.AddLink(overallLinks, baseUrl, "collection", fmt.Sprintf("/collections/%s", collectionId), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "parent", fmt.Sprintf("/collections/%s", collectionId), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "root", "/", "application/json")

	queryParts := buildQueryArray(c)
	token := c.Query("token", "")
	var queryPartsFull []string
	if token != "" {
		queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", token))
	}
	query := strings.Join(queryPartsFull, "&")
	if query != "" {
		overallLinks = stac.AddLink(overallLinks, baseUrl, "self", fmt.Sprintf("/collections/%s/items?%s", collectionId, query), "application/geo+json")
	} else {
		overallLinks = stac.AddLink(overallLinks, baseUrl, "self", fmt.Sprintf("/collections/%s/items", collectionId), "application/geo+json")
	}

	if featureCollection.Next != "" {
		queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", featureCollection.Next))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseUrl, "next", fmt.Sprintf("/collections/%s/items?%s", collectionId, query), "application/geo+json")
	}

	if featureCollection.Prev != "" {
		queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", featureCollection.Prev))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseUrl, "previous", fmt.Sprintf("/collections/%s/items?%s", collectionId, query), "application/geo+json")
	}

	return c.JSON(struct {
		Type     string                        `json:"type"`
		Context  *json.RawMessage              `json:"context"`
		Features []map[string]*json.RawMessage `json:"features"`
		Links    []stac.Link                   `json:"links"`
	}{
		Type:     "FeatureCollection",
		Context:  featureCollection.Context,
		Features: featureCollection.Features,
		Links:    overallLinks,
	})
}

func itemFromIDs(c *fiber.Ctx, ids []string) error {
	ctx := context.Background()
	baseUrl := getBaseUrl(c)
	collectionId := c.Params("collectionId")

	pool := database.GetInstance(ctx)

	// make sure the requested collection exists
	row := pool.QueryRow(ctx, "SELECT id FROM pgstac.collections WHERE id=$1", collectionId)
	var dbResult string
	if err := row.Scan(&dbResult); err != nil {
		log.Error().Err(err).Str("collectionId", collectionId).Msg("collection does not exist in database")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Error{
			Code:        stac.NotFoundError,
			Description: "could not query collections table",
		})
	}

	// do the search
	conf := json.RawMessage(`{"nohydrate": false}`)
	cql := stac.CQL{
		Collections: []string{collectionId},
		Ids:         ids,
		Limit:       10,
		Conf:        &conf,
	}

	featureCollection, err := stac.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        stac.ServerError,
			Description: "stac search returned an error",
		})
	}

	// enrich links
	for _, item := range featureCollection.Features {
		var myLinksJson json.RawMessage
		var itemId string
		var links []stac.Link

		if err := json.Unmarshal(*item["id"], &itemId); err != nil {
			log.Error().Err(err).Msg("error de-serializing id")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error de-serializing item id",
			})
		}

		if err := json.Unmarshal(*item["links"], &links); err != nil {
			log.Error().Err(err).Msg("error de-serializing link")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error de-serializing item link",
			})
		}
		for idx, link := range links {
			if link.Rel == "collection" {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseUrl, collectionId)
			}
			links[idx] = link
		}

		links = stac.AddLink(links, baseUrl, "parent", fmt.Sprintf("/collections/%s", collectionId), "application/json")
		links = stac.AddLink(links, baseUrl, "root", "/", "application/json")
		links = stac.AddLink(links, baseUrl, "self", fmt.Sprintf("/collections/%s/items/%s", collectionId, itemId), "application/geo+json")

		myLinksJson, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("error serializing links")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: "error serializing item links",
			})
		}

		item["links"] = &myLinksJson
	}

	// overall links
	overallLinks := make([]stac.Link, 0, 4)
	overallLinks = stac.AddLink(overallLinks, baseUrl, "collection", fmt.Sprintf("/collections/%s", collectionId), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "parent", fmt.Sprintf("/collections/%s", collectionId), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "root", "/", "application/json")

	queryParts := []string{fmt.Sprintf("ids=%s", strings.Join(ids, ","))}
	query := strings.Join(queryParts, "&")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "self", fmt.Sprintf("/collections/%s/items?%s", collectionId, query), "application/geo+json")

	if featureCollection.Next != "" {
		queryPartsFull := append(queryParts, fmt.Sprintf("token=%s", featureCollection.Next))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseUrl, "next", fmt.Sprintf("/collections/%s/items?%s", collectionId, query), "application/geo+json")
	}

	if featureCollection.Prev != "" {
		queryPartsFull := append(queryParts, fmt.Sprintf("token=%s", featureCollection.Prev))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseUrl, "previous", fmt.Sprintf("/collections/%s/items?%s", collectionId, query), "application/geo+json")
	}

	return c.JSON(struct {
		Type     string                        `json:"type"`
		Context  *json.RawMessage              `json:"context"`
		Features []map[string]*json.RawMessage `json:"features"`
		Links    []stac.Link                   `json:"links"`
	}{
		Type:     "FeatureCollection",
		Context:  featureCollection.Context,
		Features: featureCollection.Features,
		Links:    overallLinks,
	})
}

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
	"errors"
	"fmt"
	"strings"

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/jsonutil"
	"github.com/go-geospatial/go-stac-server/stac"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// DeleteItem deletes an item from the database
// DELETE /collections/:collectionId/items/:itemId
func DeleteItem(c *fiber.Ctx) error {
	ctx := context.Background()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT delete_item($1::text, $2::text);", itemID, collectionID); err != nil {
		log.Error().Err(err).Msg("received error while trying to delete item")
		c.Status(fiber.ErrNotFound.Code)
		if err2 := c.JSON(stac.Message{
			Code:        "DeleteItemFailed",
			Description: "cannot find item; failed to delete.",
		}); err2 != nil {
			return err2
		}
		return err
	}

	return c.JSON(stac.Message{
		Code:        "ItemDeleted",
		Description: "the item has been deleted",
	})
}

// UpdateItem updates an existing item with the provided JSON
// PUT /collections/:collectionId/items/:itemId
func UpdateItem(c *fiber.Ctx) error {
	ctx := context.Background()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	// set collectionId and and itemId from URL
	item := make(map[string]*json.RawMessage)
	if err := json.Unmarshal(c.Body(), &item); err != nil {
		log.Error().Err(err).Msg("failed to un-marshal body")
		c.Status(fiber.StatusUnprocessableEntity)
		if err2 := c.JSON(stac.Message{
			Code:        "PutItemFailed",
			Description: "failed to parse http body as JSON",
		}); err2 != nil {
			return err2
		}
		return err
	}

	// Check that the id's supplied match those from the URL. If no id's supplied populate with values from URL
	item, err := checkBodyIDAgainstURL(c, collectionID, itemID, item)
	if err != nil {
		return err
	}

	// everything checks out ... do the update ... if the attempted update fails it's because the
	// item doesn't currently exist in database so send 404

	putItem, err := json.Marshal(item)
	if err != nil {
		log.Error().Err(err).Msg("failed to serialize item")
		c.Status(fiber.StatusInternalServerError)
		if err2 := c.JSON(stac.Message{
			Code:        "ItemSerializeFailed",
			Description: "could not serialize item",
		}); err2 != nil {
			return err2
		}
		return err
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT update_item($1::text::jsonb);", putItem); err != nil {
		log.Error().Err(err).Msg("received error while trying to update item")
		c.Status(fiber.StatusNotFound)
		if err2 := c.JSON(stac.Message{
			Code:        "PutItemFailed",
			Description: fmt.Sprintf("collection %s does not contain an item with id %s", collectionID, itemID),
		}); err2 != nil {
			return err2
		}
		return err
	}

	return itemFromID(c, collectionID, itemID)
}

// PatchItem updates an item with only the specific fields provided by request body
// PATCH /collections/:collectionId/items/:itemId
func PatchItem(c *fiber.Ctx) error {
	ctx := context.Background()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	// set collectionId and and itemId from URL
	item := make(map[string]*json.RawMessage)
	if err := json.Unmarshal(c.Body(), &item); err != nil {
		log.Error().Err(err).Msg("failed to un-marshal body")
		c.Status(fiber.StatusUnprocessableEntity)
		if err2 := c.JSON(stac.Message{
			Code:        "PatchItemFailed",
			Description: "failed to parse http body as JSON",
		}); err2 != nil {
			return err2
		}
		return err
	}

	// Check that the id's supplied match those from the URL. If no id's supplied populate with values from URL
	item, err := checkBodyIDAgainstURL(c, collectionID, itemID, item)
	if err != nil {
		return err
	}

	// get the item from the database
	pool := database.GetInstance(ctx)
	var dbItemRaw string
	if err := pool.QueryRow(ctx, "SELECT get_item FROM get_item($1::text, $2::text);", itemID, collectionID).Scan(&dbItemRaw); err != nil {
		log.Error().Err(err).Msg("failed load item from database")
		c.Status(fiber.StatusNotFound)
		if err2 := c.JSON(stac.Message{
			Code:        "ItemNotFound",
			Description: "item was not found",
		}); err2 != nil {
			return err2
		}
		return err
	}

	patchItem, err := json.Marshal(item)
	if err != nil {
		log.Error().Err(err).Msg("failed to serialize item")
		c.Status(fiber.StatusInternalServerError)
		if err2 := c.JSON(stac.Message{
			Code:        "ItemSerializeFailed",
			Description: "could not serialize item",
		}); err2 != nil {
			return err2
		}
		return err
	}

	// merge items
	mergedItem, err := jsonutil.Merge(patchItem, []byte(dbItemRaw))
	if err != nil {
		log.Error().Err(err).Msg("failed to merge items")
		c.Status(fiber.StatusInternalServerError)
		if err2 := c.JSON(stac.Message{
			Code:        "MergeItem",
			Description: "failed to merge items",
		}); err2 != nil {
			return err2
		}
		return err
	}

	// upate database
	if _, err := pool.Exec(ctx, "SELECT update_item($1::text::jsonb);", mergedItem); err != nil {
		log.Error().Err(err).Msg("received error while trying to update item")
		c.Status(fiber.StatusNotFound)
		if err2 := c.JSON(stac.Message{
			Code:        "PutItemFailed",
			Description: fmt.Sprintf("collection %s does not contain an item with id %s", collectionID, itemID),
		}); err2 != nil {
			return err2
		}
		return err
	}

	return itemFromID(c, collectionID, itemID)
}

// CreateItems creates a new collection in the database
// POST /collections/:collectionId/items
func CreateItems(c *fiber.Ctx) error {
	// validate passed JSON
	itemsRaw := c.Body()
	items := make(map[string]*json.RawMessage)

	if err := json.Unmarshal(itemsRaw, &items); err != nil {
		log.Error().Err(err).Str("RequestBody", string(itemsRaw)).Msg("cannot unmarshal body to items")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Message{
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
		return c.JSON(stac.Message{
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
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "invalid geojson type - must be one of 'Feature' or 'FeatureCollection'",
		})
	}
}

func createFeature(c *fiber.Ctx, items map[string]*json.RawMessage, itemsRaw []byte) error {
	ctx := context.Background()
	collectionID := c.Params("collectionId")
	var err error

	// validate collection matches the expected collection
	if err = stac.ValidateCollectionIDsMatch(c, items, collectionID); err != nil {
		return err
	}

	// validate item itemID
	var itemID string
	if itemID, err = stac.ValidateID(c, items); err != nil {
		return err
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal items to JSON")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "failed to marshal JSON for items",
		})
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT create_item($1::text::jsonb)", itemsJSON); err != nil {
		log.Error().Err(err).Str("id", itemID).Str("raw", string(itemsRaw)).Msg("failed to create item")
		c.Status(fiber.ErrConflict.Code)
		return c.JSON(stac.Message{
			Code:        "CreateItemFailed",
			Description: "failed to create item",
		})
	}

	return itemFromID(c, collectionID, itemID)
}

func createFeatureCollection(c *fiber.Ctx, items map[string]*json.RawMessage, itemsRaw []byte) error {
	ctx := context.Background()
	collectionID := c.Params("collectionId")

	// for each feature validate collection matches the expected collection
	var featuresRaw *json.RawMessage
	var ok bool
	if featuresRaw, ok = items["features"]; !ok {
		log.Error().Str("raw", string(itemsRaw)).Msg("failed to get features - object invalid")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "objects of type 'FeatureCollection' must have a 'features' field",
		})
	}

	var features []map[string]*json.RawMessage
	if err := json.Unmarshal(*featuresRaw, &features); err != nil {
		log.Error().Err(err).Str("raw", string(itemsRaw)).Msg("unmarshal geojson features failed")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "objects of type 'FeatureCollection' must have a valid 'features' field",
		})
	}

	itemIds := make([]string, len(features))
	for idx, feature := range features {
		if err := stac.ValidateCollectionIDsMatch(c, feature, collectionID); err != nil {
			log.Error().Int("FeatureIndex", idx).Msg("failed collection ID match validation")
			return err
		}
		itemID, err := stac.ValidateID(c, feature)
		if err != nil {
			log.Error().Int("FeatureIndex", idx).Msg("failed ID validation")
			return err
		}
		itemIds[idx] = itemID
	}

	// validation has passed, create items
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal items to JSON")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "failed to marshal JSON for items",
		})
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT create_items($1::text::jsonb)", itemsJSON); err != nil {
		log.Error().Err(err).Strs("id", itemIds).Str("raw", string(itemsRaw)).Msg("failed to create item")
		c.Status(fiber.ErrConflict.Code)
		return c.JSON(stac.Message{
			Code:        "CreateItemFailed",
			Description: "failed to create item",
		})
	}

	return itemFromIDs(c, itemIds)
}

// Item returns details of a specific item
// GET /collections/:collectionId/items/:itemId
func Item(c *fiber.Ctx) error {
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	return itemFromID(c, collectionID, itemID)
}

func itemFromID(c *fiber.Ctx, collectionID string, itemID string) error {
	ctx := context.Background()
	baseURL := getBaseURL(c)

	pool := database.GetInstance(ctx)

	// make sure the requested collection exists
	row := pool.QueryRow(ctx, "SELECT id FROM pgstac.collections WHERE id=$1", collectionID)
	var dbResult string
	if err := row.Scan(&dbResult); err != nil {
		log.Error().Err(err).Str("collectionId", collectionID).Msg("collection does not exist in database")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Message{
			Code:        stac.NotFoundError,
			Description: "could not query collections table",
		})
	}

	// create CQL search criteria
	conf := json.RawMessage(`{"nohydrate": false}`)
	cql := stac.CQL{
		Collections: []string{collectionID},
		Ids:         []string{itemID},
		Conf:        &conf,
		Limit:       1,
	}

	// do the search
	featureCollection, err := stac.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Message{
			Code:        stac.ServerError,
			Description: "stac search returned an error",
		})
	}

	// enrich links
	var myItem map[string]*json.RawMessage
	for _, item := range featureCollection.Features {
		var myLinksJSON json.RawMessage
		var itemID string
		var links []stac.Link

		if err := json.Unmarshal(*item["id"], &itemID); err != nil {
			log.Error().Err(err).Msg("error de-serializing id")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item id",
			})
		}

		if err := json.Unmarshal(*item["links"], &links); err != nil {
			log.Error().Err(err).Msg("error de-serializing link")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item link",
			})
		}
		for idx, link := range links {
			if link.Rel == stac.CollectionKey {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseURL, collectionID)
			}
			links[idx] = link
		}

		links = stac.AddLink(links, baseURL, "parent", fmt.Sprintf("/collections/%s", collectionID), "application/json")
		links = stac.AddLink(links, baseURL, "root", "/", "application/json")
		links = stac.AddLink(links, baseURL, "self", fmt.Sprintf("/collections/%s/items/%s", collectionID, itemID), "application/geo+json")

		myLinksJSON, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("error serializing links")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error serializing item links",
			})
		}

		item["links"] = &myLinksJSON
		myItem = item
	}

	return c.JSON(myItem)
}

// Items returns a list of items in a collection
// GET /collections/:collectionId/items
func Items(c *fiber.Ctx) error {
	ctx := context.Background()
	baseURL := getBaseURL(c)
	collectionID := c.Params("collectionId")

	pool := database.GetInstance(ctx)

	// make sure the requested collection exists
	row := pool.QueryRow(ctx, "SELECT id FROM pgstac.collections WHERE id=$1", collectionID)
	var dbResult string
	if err := row.Scan(&dbResult); err != nil {
		log.Error().Err(err).Str("collectionId", collectionID).Msg("collection does not exist in database")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Message{
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
	cql.Collections = []string{collectionID}
	featureCollection, err := stac.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Message{
			Code:        stac.ServerError,
			Description: "stac search returned an error",
		})
	}

	// enrich links
	for _, item := range featureCollection.Features {
		var myLinksJSON json.RawMessage
		var itemID string
		var links []stac.Link

		if err := json.Unmarshal(*item["id"], &itemID); err != nil {
			log.Error().Err(err).Msg("error de-serializing id")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item id",
			})
		}

		if err := json.Unmarshal(*item["links"], &links); err != nil {
			log.Error().Err(err).Msg("error de-serializing link")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item link",
			})
		}
		for idx, link := range links {
			if link.Rel == stac.CollectionKey {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseURL, collectionID)
			}
			links[idx] = link
		}

		links = stac.AddLink(links, baseURL, "parent", fmt.Sprintf("/collections/%s", collectionID), "application/json")
		links = stac.AddLink(links, baseURL, "root", "/", "application/json")
		links = stac.AddLink(links, baseURL, "self", fmt.Sprintf("/collections/%s/items/%s", collectionID, itemID), "application/geo+json")

		myLinksJSON, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("error serializing links")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error serializing item links",
			})
		}

		item["links"] = &myLinksJSON
	}

	// overall links
	overallLinks := make([]stac.Link, 0, 4)
	overallLinks = stac.AddLink(overallLinks, baseURL, "collection", fmt.Sprintf("/collections/%s", collectionID), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "parent", fmt.Sprintf("/collections/%s", collectionID), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "root", "/", "application/json")

	queryParts := buildQueryArray(c)
	token := c.Query("token", "")
	var queryPartsFull []string
	if token != "" {
		queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", token))
	}
	query := strings.Join(queryPartsFull, "&")
	if query != "" {
		overallLinks = stac.AddLink(overallLinks, baseURL, "self", fmt.Sprintf("/collections/%s/items?%s", collectionID, query), "application/geo+json")
	} else {
		overallLinks = stac.AddLink(overallLinks, baseURL, "self", fmt.Sprintf("/collections/%s/items", collectionID), "application/geo+json")
	}

	if featureCollection.Next != "" {
		queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", featureCollection.Next))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseURL, "next", fmt.Sprintf("/collections/%s/items?%s", collectionID, query), "application/geo+json")
	}

	if featureCollection.Prev != "" {
		queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", featureCollection.Prev))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseURL, "previous", fmt.Sprintf("/collections/%s/items?%s", collectionID, query), "application/geo+json")
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
	baseURL := getBaseURL(c)
	collectionID := c.Params("collectionId")

	pool := database.GetInstance(ctx)

	// make sure the requested collection exists
	row := pool.QueryRow(ctx, "SELECT id FROM pgstac.collections WHERE id=$1", collectionID)
	var dbResult string
	if err := row.Scan(&dbResult); err != nil {
		log.Error().Err(err).Str("collectionId", collectionID).Msg("collection does not exist in database")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Message{
			Code:        stac.NotFoundError,
			Description: "could not query collections table",
		})
	}

	// do the search
	conf := json.RawMessage(`{"nohydrate": false}`)
	cql := stac.CQL{
		Collections: []string{collectionID},
		Ids:         ids,
		Limit:       10,
		Conf:        &conf,
	}

	featureCollection, err := stac.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Message{
			Code:        stac.ServerError,
			Description: "stac search returned an error",
		})
	}

	// enrich links
	for _, item := range featureCollection.Features {
		var myLinksJSON json.RawMessage
		var itemID string
		var links []stac.Link

		if err := json.Unmarshal(*item["id"], &itemID); err != nil {
			log.Error().Err(err).Msg("error de-serializing id")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item id",
			})
		}

		if err := json.Unmarshal(*item["links"], &links); err != nil {
			log.Error().Err(err).Msg("error de-serializing link")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item link",
			})
		}
		for idx, link := range links {
			if link.Rel == "collection" {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseURL, collectionID)
			}
			links[idx] = link
		}

		links = stac.AddLink(links, baseURL, "parent", fmt.Sprintf("/collections/%s", collectionID), "application/json")
		links = stac.AddLink(links, baseURL, "root", "/", "application/json")
		links = stac.AddLink(links, baseURL, "self", fmt.Sprintf("/collections/%s/items/%s", collectionID, itemID), "application/geo+json")

		myLinksJSON, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("error serializing links")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error serializing item links",
			})
		}

		item["links"] = &myLinksJSON
	}

	// overall links
	overallLinks := make([]stac.Link, 0, 4)
	overallLinks = stac.AddLink(overallLinks, baseURL, stac.CollectionKey, fmt.Sprintf("/collections/%s", collectionID), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "parent", fmt.Sprintf("/collections/%s", collectionID), "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "root", "/", "application/json")

	queryParts := []string{fmt.Sprintf("ids=%s", strings.Join(ids, ","))}
	query := strings.Join(queryParts, "&")
	overallLinks = stac.AddLink(overallLinks, baseURL, "self", fmt.Sprintf("/collections/%s/items?%s", collectionID, query), "application/geo+json")

	if featureCollection.Next != "" {
		queryPartsFull := append(queryParts, fmt.Sprintf("token=%s", featureCollection.Next))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseURL, "next", fmt.Sprintf("/collections/%s/items?%s", collectionID, query), "application/geo+json")
	}

	if featureCollection.Prev != "" {
		queryPartsFull := append(queryParts, fmt.Sprintf("token=%s", featureCollection.Prev))
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseURL, "previous", fmt.Sprintf("/collections/%s/items?%s", collectionID, query), "application/geo+json")
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

func checkBodyIDAgainstURL(c *fiber.Ctx, collectionID string, itemID string, item map[string]*json.RawMessage) (map[string]*json.RawMessage, error) {
	// check that itemId and collectionId are set and match those provided in the URL
	if bodyItemID, ok := item["id"]; !ok {
		var itemIDSerialized json.RawMessage
		itemIDSerialized, err := json.Marshal(itemID)
		if err != nil {
			log.Error().Msg("body does not include item id")
			c.Status(fiber.StatusBadRequest)
			if err2 := c.JSON(stac.Message{
				Code:        "ModifyItemFailed",
				Description: "item must include an `id` field with the item id",
			}); err2 != nil {
				return nil, err2
			}
			return nil, err
		}
		item["id"] = &itemIDSerialized
	} else if string(*bodyItemID) != itemID {
		log.Error().Str("BodyItemId", string(*bodyItemID)).Str("URLItemId", itemID).Msg("PUT body item id does not match URL item id")
		c.Status(fiber.StatusBadRequest)
		if err2 := c.JSON(stac.Message{
			Code:        "ModifyItemFailed",
			Description: "item must include an `id` field with the item id that matches the URL item id",
		}); err2 != nil {
			return nil, err2
		}
		return nil, errors.New("specified item id and url item id do not match")
	}

	if bodyCollectionID, ok := item["collection"]; !ok {
		var collectionSerialized json.RawMessage
		collectionSerialized, err := json.Marshal(collectionID)
		if err != nil {
			log.Error().Err(err).Msg("could not serialize collection id")
			c.Status(fiber.StatusInternalServerError)
			if err2 := c.JSON(stac.Message{
				Code:        "ModifyItemFailed",
				Description: fmt.Sprintf("serialize collection id failed: %s", err.Error()),
			}); err2 != nil {
				return nil, err2
			}
			return nil, err
		}
		item[stac.CollectionKey] = &collectionSerialized
	} else if string(*bodyCollectionID) != collectionID {
		log.Error().Str("BodyCollectionId", string(*bodyCollectionID)).Str("URLCollectionId", itemID).Msg("PUT body collection id does not match URL collection id")
		c.Status(fiber.StatusBadRequest)
		if err2 := c.JSON(stac.Message{
			Code:        "ModifyItemFailed",
			Description: "item must include a `collection` field with the coillection id that matches the URL collection id",
		}); err2 != nil {
			return nil, err2
		}
		return nil, errors.New("specified collection id and url collection id do not match")
	}

	return item, nil
}

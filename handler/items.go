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
	"encoding/json"
	"fmt"

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/stac"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Item returns details of a specific item
func Item(c *fiber.Ctx) error {
	ctx := context.Background()
	baseUrl := getBaseUrl(c)

	// get params
	collectionId := c.Params("collectionId")
	itemId := c.Params("itemId")

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
func Items(c *fiber.Ctx) error {
	ctx := context.Background()
	baseUrl := getBaseUrl(c)

	// get params
	collectionId := c.Params("id")
	limitStr := c.Query("limit", "10")
	bboxStr := c.Query("bbox", "")
	dateStr := c.Query("datetime", "")
	token := c.Query("token", "")

	// parse limit
	limit, err := parseLimit(c, limitStr)
	if err != nil {
		// response and logging handled by parseLimit
		return err
	}

	// parse bbox
	bbox, err := parseBbox(c, bboxStr)
	if err != nil {
		// response and logging handled by parseBbox
		return err
	}

	// parse date string (must be RFC 3339)
	if err := parseRFC3339Date(c, dateStr); err != nil {
		// http response and logging handled by parseRFC3339Date
		return err
	}

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
		Limit:       limit,
		DateTime:    dateStr,
		Token:       token,
		Conf:        &conf,
	}

	if bboxStr != "" {
		cql.Bbox = bbox
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
	overallLinks = stac.AddLink(overallLinks, baseUrl, "self", fmt.Sprintf("/collections/%s/items", collectionId), "application/geo+json")

	if featureCollection.Next != "" {
		overallLinks = stac.AddLink(overallLinks, baseUrl, "next", fmt.Sprintf("/collections/%s/items?token=%s", collectionId, featureCollection.Next), "application/geo+json")
	}

	if featureCollection.Prev != "" {
		overallLinks = stac.AddLink(overallLinks, baseUrl, "previous", fmt.Sprintf("/collections/%s/items?token=%s", collectionId, featureCollection.Next), "application/geo+json")
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

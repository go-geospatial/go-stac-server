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
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Collection returns details of a specific collection
func Collection(c *fiber.Ctx) error {
	ctx := context.Background()
	collectionId := c.Params("id")
	baseUrl := getBaseUrl(c)

	// get a list of all collections
	pool := database.GetInstance(ctx)
	row := pool.QueryRow(ctx, "SELECT content FROM pgstac.collections WHERE id=$1", collectionId)

	collection := make(map[string]*json.RawMessage, 20)
	var rawCollection string
	err := row.Scan(&rawCollection)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			log.Error().Err(err).Str("collectionId", collectionId).Msg("collection not found")
			c.Status(fiber.ErrNotFound.Code)
			return c.JSON(stac.Error{
				Code:        "404",
				Description: "collection not found",
			})
		default:
			log.Error().Err(err).Msg("could not scan collection id and title")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        database.QueryErrorCode,
				Description: "could not serialize data from collections table",
			})
		}
	}

	// un-marshal to map
	if err := json.Unmarshal([]byte(rawCollection), &collection); err != nil {
		log.Error().Err(err).Msg("collection JSON unmarshal failed")
		c.Status(fiber.ErrInternalServerError.Code)
		c.JSON(stac.Error{
			Code:        stac.JSONParsingError,
			Description: "unable to un-marshal collection object JSON",
		})
	}

	// un-marshal links
	links := make([]stac.Link, 0, 5)
	if rawLinks, ok := collection["links"]; ok {
		if err := json.Unmarshal(*rawLinks, &links); err != nil {
			log.Error().Err(err).Msg("collection JSON unmarshal failed")
			c.Status(fiber.ErrInternalServerError.Code)
			c.JSON(stac.Error{
				Code:        stac.JSONParsingError,
				Description: "unable to un-marshal collection links JSON",
			})
		}
	}

	// enrich links with self, root, parent, and items references
	collectionsEndpoint := fmt.Sprintf("/collections/%s", collectionId)
	links = stac.AddLink(links, baseUrl, "self", collectionsEndpoint, "application/json")
	links = stac.AddLink(links, baseUrl, "root", "/", "application/json")
	links = stac.AddLink(links, baseUrl, "parent", "/", "application/json")
	links = stac.AddLink(links, baseUrl, "items", fmt.Sprintf("%s/items", collectionsEndpoint), "application/geo+json")

	var serializedLinks json.RawMessage
	serializedLinks, err = json.Marshal(links)
	if err != nil {
		log.Error().Err(err).Msg("collection links JSON marshal failed")
		c.Status(fiber.ErrInternalServerError.Code)
		c.JSON(stac.Error{
			Code:        stac.JSONParsingError,
			Description: "unable to marshal collection links to JSON",
		})
	}
	collection["links"] = &serializedLinks

	return c.JSON(collection)
}

// Collections returns a list of collections managed by this STAC server
func Collections(c *fiber.Ctx) error {
	ctx := context.Background()
	baseUrl := getBaseUrl(c)

	collections := make([]*json.RawMessage, 0, 10)

	// get a list of all collections
	pool := database.GetInstance(ctx)
	rows, err := pool.Query(ctx, "SELECT id, content FROM pgstac.collections ORDER BY id")
	if err != nil {
		log.Error().Err(err).Msg("error querying collections for list collections response")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        database.QueryErrorCode,
			Description: "could not query collections table",
		})
	}
	defer rows.Close()
	for rows.Next() {
		collection := make(map[string]*json.RawMessage, 20)
		var rawCollection string
		var collectionId string
		err := rows.Scan(&collectionId, &rawCollection)
		if err != nil {
			log.Error().Err(err).Msg("could not scan collection id and title")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        database.QueryErrorCode,
				Description: "could not serialize data from collections table",
			})
		}

		// un-marshal to map
		if err := json.Unmarshal([]byte(rawCollection), &collection); err != nil {
			log.Error().Err(err).Msg("collection JSON unmarshal failed")
			c.Status(fiber.ErrInternalServerError.Code)
			c.JSON(stac.Error{
				Code:        stac.JSONParsingError,
				Description: "unable to un-marshal collection object JSON",
			})
		}

		// un-marshal links
		links := make([]stac.Link, 0, 5)
		if rawLinks, ok := collection["links"]; ok {
			if err := json.Unmarshal(*rawLinks, &links); err != nil {
				log.Error().Err(err).Msg("collection JSON unmarshal failed")
				c.Status(fiber.ErrInternalServerError.Code)
				c.JSON(stac.Error{
					Code:        stac.JSONParsingError,
					Description: "unable to un-marshal collection links JSON",
				})
			}
		}

		// enrich links with self, root, parent, and items references
		collectionsEndpoint := fmt.Sprintf("/collections/%s", collectionId)
		links = stac.AddLink(links, baseUrl, "self", collectionsEndpoint, "application/json")
		links = stac.AddLink(links, baseUrl, "root", "/", "application/json")
		links = stac.AddLink(links, baseUrl, "parent", "/", "application/json")
		links = stac.AddLink(links, baseUrl, "items", fmt.Sprintf("%s/items", collectionsEndpoint), "application/geo+json")

		var serializedLinks json.RawMessage
		serializedLinks, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("collection links JSON marshal failed")
			c.Status(fiber.ErrInternalServerError.Code)
			c.JSON(stac.Error{
				Code:        stac.JSONParsingError,
				Description: "unable to marshal collection links to JSON",
			})
		}
		collection["links"] = &serializedLinks

		var serializedCollection json.RawMessage
		serializedCollection, err = json.Marshal(collection)
		if err != nil {
			log.Error().Err(err).Msg("collection JSON marshal failed")
			c.Status(fiber.ErrInternalServerError.Code)
			c.JSON(stac.Error{
				Code:        stac.JSONParsingError,
				Description: "unable to marshal collection to JSON",
			})
		}
		collections = append(collections, &serializedCollection)
	}

	overallLinks := make([]stac.Link, 0, 3)
	overallLinks = stac.AddLink(overallLinks, baseUrl, "self", "/collections", "application/json")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "root", "/", "application/json")
	overallLinks = stac.AddLink(overallLinks, baseUrl, "parent", "/", "application/json")

	return c.JSON(struct {
		Collections []*json.RawMessage `json:"collections"`
		Links       []stac.Link        `json:"links"`
	}{
		Collections: collections,
		Links:       overallLinks,
	})
}

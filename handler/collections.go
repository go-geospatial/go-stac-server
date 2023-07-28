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

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/stac"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// ModifyCollection creates a new collection in the database
// POST /collections
// PUT /collections
func ModifyCollection(c *fiber.Ctx) error {
	ctx := context.Background()

	// validate passed JSON
	collectionRaw := c.Body()
	collection := make(map[string]*json.RawMessage)

	if err := json.Unmarshal(collectionRaw, &collection); err != nil {
		log.Error().Err(err).Str("RequestBody", string(collectionRaw)).Msg("cannot unmarshal provided JSON in CreateCollection")
		c.Status(fiber.StatusBadRequest)
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "JSON parse failed; collection must be a valid JSON object",
		})
	}

	var id string
	var err error
	if id, err = stac.ValidateID(c, collection); err != nil {
		return err
	}

	collectionJSON, err := json.Marshal(collection)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal collection to JSON")
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "failed to marshal JSON for collection",
		})
	}

	query := "SELECT create_collection($1::text::jsonb)"
	if c.Method() == "PUT" {
		log.Info().Msg("updating collection")
		query = "SELECT update_collection($1::text::jsonb)"
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, query, collectionJSON); err != nil {
		log.Error().Err(err).Str("id", id).Str("raw", string(collectionRaw)).Msg("failed to create collection")
		c.Status(fiber.StatusBadRequest)
		return c.JSON(stac.Message{
			Code:        "CreateCollectionFailed",
			Description: "failed to create collection",
		})
	}

	return collectionFromID(c, id)
}

// DeleteCollection creates a new collection in the database
// DELETE /collections
func DeleteCollection(c *fiber.Ctx) error {
	ctx := context.Background()
	collectionID := c.Params("collectionId")

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT delete_collection($1::text)", collectionID); err != nil {
		log.Error().Err(err).Str("id", collectionID).Msg("collection not found")
		c.Status(fiber.ErrNotFound.Code)
		return c.JSON(stac.Message{
			Code:        stac.NotFoundError,
			Description: "collection not found",
		})
	}

	// NOTE: we use the error struct here for convenience because it has a suitable structure for the response
	return c.JSON(stac.Message{
		Code:        "CollectionDeleted",
		Description: "the collection was successfully deleted",
	})
}

// Collection returns details of a specific collection
// GET /collections/:collectionId/
func Collection(c *fiber.Ctx) error {
	collectionID := c.Params("collectionId")
	return collectionFromID(c, collectionID)
}

func collectionFromID(c *fiber.Ctx, collectionID string) error {
	ctx := context.Background()
	baseURL := getBaseURL(c)

	// get a list of all collections
	pool := database.GetInstance(ctx)
	row := pool.QueryRow(ctx, "SELECT get_collection FROM pgstac.get_collection($1)", collectionID)

	collection := make(map[string]*json.RawMessage, 20)
	var rawCollection string
	err := row.Scan(&rawCollection)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Error().Err(err).Str("collectionId", collectionID).Msg("collection not found")
			c.Status(fiber.ErrNotFound.Code)
			return c.JSON(stac.Message{
				Code:        "404",
				Description: "collection not found",
			})
		}

		// pgstac returns a row even if the collection doesn't exist.
		log.Error().Str("collection", collectionID).Msg("collection not found")
		c.Status(fiber.StatusNotFound)
		return c.JSON(stac.Message{
			Code:        stac.NotFoundError,
			Description: fmt.Sprintf("collection '%s' not found", collectionID),
		})
	}

	// un-marshal to map
	if err := json.Unmarshal([]byte(rawCollection), &collection); err != nil {
		log.Error().Err(err).Msg("collection JSON unmarshal failed")
		c.Status(fiber.StatusInternalServerError)
		_ = c.JSON(stac.Message{
			Code:        stac.JSONParsingError,
			Description: "unable to un-marshal collection object JSON",
		})
		return err
	}

	// un-marshal links
	links := make([]stac.Link, 0, 5)
	if rawLinks, ok := collection["links"]; ok {
		if err := json.Unmarshal(*rawLinks, &links); err != nil {
			log.Error().Err(err).Msg("collection JSON unmarshal failed")
			c.Status(fiber.StatusInternalServerError)
			_ = c.JSON(stac.Message{
				Code:        stac.JSONParsingError,
				Description: "unable to un-marshal collection links JSON",
			})
			return err
		}
	}

	// enrich links with self, root, parent, and items references
	collectionsEndpoint := fmt.Sprintf("/collections/%s", collectionID)
	links = stac.AddLink(links, baseURL, "self", collectionsEndpoint, "application/json")
	links = stac.AddLink(links, baseURL, "root", "/", "application/json")
	links = stac.AddLink(links, baseURL, "parent", "/", "application/json")
	links = stac.AddLink(links, baseURL, "items", fmt.Sprintf("%s/items", collectionsEndpoint), "application/geo+json")

	var serializedLinks json.RawMessage
	serializedLinks, err = json.Marshal(links)
	if err != nil {
		log.Error().Err(err).Msg("collection links JSON marshal failed")
		c.Status(fiber.StatusInternalServerError)
		_ = c.JSON(stac.Message{
			Code:        stac.JSONParsingError,
			Description: "unable to marshal collection links to JSON",
		})
		return err
	}
	collection["links"] = &serializedLinks

	collectionType := json.RawMessage(`"Collection"`)
	collection["type"] = &collectionType

	return c.JSON(collection)
}

// Collections returns a list of collections managed by this STAC server
// GET /collections/
func Collections(c *fiber.Ctx) error {
	ctx := context.Background()
	baseURL := getBaseURL(c)

	collections := make([]*json.RawMessage, 0, 10)

	// get a list of all collections
	pool := database.GetInstance(ctx)
	rows, err := pool.Query(ctx, "SELECT id, content FROM pgstac.collections ORDER BY id")
	if err != nil {
		log.Error().Err(err).Msg("error querying collections for list collections response")
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(stac.Message{
			Code:        database.QueryErrorCode,
			Description: "could not query collections table",
		})
	}
	defer rows.Close()
	for rows.Next() {
		collection := make(map[string]*json.RawMessage, 20)
		var rawCollection string
		var collectionID string
		err := rows.Scan(&collectionID, &rawCollection)
		if err != nil {
			log.Error().Err(err).Msg("could not scan collection id and title")
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(stac.Message{
				Code:        database.QueryErrorCode,
				Description: "could not serialize data from collections table",
			})
		}

		// un-marshal to map
		if err := json.Unmarshal([]byte(rawCollection), &collection); err != nil {
			log.Error().Err(err).Msg("collection JSON unmarshal failed")
			c.Status(fiber.StatusInternalServerError)
			_ = c.JSON(stac.Message{
				Code:        stac.JSONParsingError,
				Description: "unable to un-marshal collection object JSON",
			})
			return err
		}

		// un-marshal links
		links := make([]stac.Link, 0, 5)
		if rawLinks, ok := collection["links"]; ok {
			if err := json.Unmarshal(*rawLinks, &links); err != nil {
				log.Error().Err(err).Msg("collection JSON unmarshal failed")
				c.Status(fiber.StatusInternalServerError)
				_ = c.JSON(stac.Message{
					Code:        stac.JSONParsingError,
					Description: "unable to un-marshal collection links JSON",
				})
				return err
			}
		}

		// enrich links with self, root, parent, and items references
		collectionsEndpoint := fmt.Sprintf("/collections/%s", collectionID)
		links = stac.AddLink(links, baseURL, "self", collectionsEndpoint, "application/json")
		links = stac.AddLink(links, baseURL, "root", "/", "application/json")
		links = stac.AddLink(links, baseURL, "parent", "/", "application/json")
		links = stac.AddLink(links, baseURL, "items", fmt.Sprintf("%s/items", collectionsEndpoint), "application/geo+json")

		var serializedLinks json.RawMessage
		serializedLinks, err = json.Marshal(links)
		if err != nil {
			log.Error().Err(err).Msg("collection links JSON marshal failed")
			c.Status(fiber.StatusInternalServerError)
			_ = c.JSON(stac.Message{
				Code:        stac.JSONParsingError,
				Description: "unable to marshal collection links to JSON",
			})
			return err
		}
		collection["links"] = &serializedLinks

		collectionType := json.RawMessage(`"Collection"`)
		collection["type"] = &collectionType

		var serializedCollection json.RawMessage
		serializedCollection, err = json.Marshal(collection)
		if err != nil {
			log.Error().Err(err).Msg("collection JSON marshal failed")
			c.Status(fiber.StatusInternalServerError)
			_ = c.JSON(stac.Message{
				Code:        stac.JSONParsingError,
				Description: "unable to marshal collection to JSON",
			})
			return err
		}
		collections = append(collections, &serializedCollection)
	}

	overallLinks := make([]stac.Link, 0, 3)
	overallLinks = stac.AddLink(overallLinks, baseURL, "self", "/collections", "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "root", "/", "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "parent", "/", "application/json")

	return c.JSON(struct {
		Collections []*json.RawMessage `json:"collections"`
		Links       []stac.Link        `json:"links"`
	}{
		Collections: collections,
		Links:       overallLinks,
	})
}

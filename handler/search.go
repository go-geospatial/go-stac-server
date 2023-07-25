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
	"fmt"
	"strings"

	"github.com/go-geospatial/go-stac-server/stac"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Item returns details of a specific item
// GET /search
// POST /search
func Search(c *fiber.Ctx) error {
	baseURL := getBaseURL(c)
	token := c.Query("token", "")

	var cql stac.CQL
	var err error
	switch c.Method() {
	case "GET":
		cql, err = getCQLFromQuery(c)
		if err != nil {
			// note http response and logging handled by getCQLFromQuery
			return err
		}
	case "POST":
		cql, err = getCQLFromBody(c)
		if err != nil {
			// note http response and logging handled by getCQLFromBody
			return err
		}
	default:
		log.Fatal().Str("Method", c.Method()).Msg("mis-configured routes - unsupported method")
	}

	if token != "" {
		cql.Token = token
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

		var collectionID string
		if err := json.Unmarshal(*item["collection"], &collectionID); err != nil {
			log.Error().Err(err).Msg("error de-serializing collectionId")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error de-serializing item collectionId",
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
	overallLinks := make([]stac.Link, 0, 5)
	overallLinks = stac.AddLink(overallLinks, baseURL, "parent", "/", "application/json")
	overallLinks = stac.AddLink(overallLinks, baseURL, "root", "/", "application/json")

	switch c.Method() {
	case "GET":
		queryParts := buildQueryArray(c)
		token := c.Query("token", "")
		var queryPartsFull []string
		if token != "" {
			queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", token))
		}
		query := strings.Join(queryPartsFull, "&")
		overallLinks = stac.AddLink(overallLinks, baseURL, "self", fmt.Sprintf("/search?%s", query), "application/geo+json")

		if featureCollection.Next != "" {
			queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", featureCollection.Next))
			query := strings.Join(queryPartsFull, "&")
			overallLinks = stac.AddLink(overallLinks, baseURL, "next", fmt.Sprintf("/search?%s", query), "application/geo+json")
		}
		if featureCollection.Prev != "" {
			queryPartsFull = append(queryParts, fmt.Sprintf("token=%s", featureCollection.Prev))
			query := strings.Join(queryPartsFull, "&")
			overallLinks = stac.AddLink(overallLinks, baseURL, "previous", fmt.Sprintf("/search?%s", query), "application/geo+json")
		}
	case "POST":
		var jsonRaw json.RawMessage
		if jsonRaw, err = json.Marshal(cql); err != nil {
			log.Error().Err(err).Msg("error serializing cql")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "error serializing cql",
			})
		}
		overallLinks = stac.AddLinkPost(overallLinks, baseURL, "self", "/search", "application/geo+json", &jsonRaw)

		if featureCollection.Next != "" {
			var jsonRaw json.RawMessage
			cql.Token = featureCollection.Next
			if jsonRaw, err = json.Marshal(cql); err != nil {
				log.Error().Err(err).Msg("error serializing cql")
				c.Status(fiber.ErrInternalServerError.Code)
				return c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "error serializing cql",
				})
			}
			overallLinks = stac.AddLinkPost(overallLinks, baseURL, "next", "/search", "application/geo+json", &jsonRaw)
		}
		if featureCollection.Prev != "" {
			var jsonRaw json.RawMessage
			cql.Token = featureCollection.Prev
			if jsonRaw, err = json.Marshal(cql); err != nil {
				log.Error().Err(err).Msg("error serializing cql")
				c.Status(fiber.ErrInternalServerError.Code)
				return c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "error serializing cql",
				})
			}
			overallLinks = stac.AddLinkPost(overallLinks, baseURL, "previous", "/search", "application/geo+json", &jsonRaw)
		}
	default:
		log.Fatal().Str("Method", c.Method()).Msg("mis-configured routes - unsupported method")
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

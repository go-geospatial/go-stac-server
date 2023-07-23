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

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/stac"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func Catalog(c *fiber.Ctx) error {
	ctx := context.Background()

	baseUrl := getBaseUrl(c)
	self := fmt.Sprintf("%s/api/stac/v1", baseUrl)
	links := make([]stac.Link, 0, 100)
	links = append(links, stac.Link{
		Rel:  "self",
		Type: "application/json",
		Href: self,
	})
	links = append(links, stac.Link{
		Rel:  "root",
		Type: "application/json",
		Href: self,
	})
	links = append(links, stac.Link{
		Rel:  "data",
		Type: "application/json",
		Href: fmt.Sprintf("%s/collections", self),
	})
	links = append(links, stac.Link{
		Rel:   "conformance",
		Type:  "application/json",
		Title: "STAC/WFS3 conformance classes implemented by this server",
		Href:  fmt.Sprintf("%s/conformance", self),
	})
	links = append(links, stac.Link{
		Rel:    "search",
		Type:   "application/geo+json",
		Title:  "STAC search",
		Href:   fmt.Sprintf("%s/search", self),
		Method: "GET",
	})
	links = append(links, stac.Link{
		Rel:    "search",
		Type:   "application/geo+json",
		Title:  "STAC search",
		Href:   fmt.Sprintf("%s/search", self),
		Method: "POST",
	})
	links = append(links, stac.Link{
		Rel:   "service-desc",
		Type:  "application/vnd.oai.openapi+json;version=3.0",
		Title: "OpenAPI service description",
		Href:  fmt.Sprintf("%s/openapi.json", baseUrl),
	})
	links = append(links, stac.Link{
		Rel:   "service-doc",
		Type:  "text/html",
		Title: "OpenAPI service documentation",
		Href:  fmt.Sprintf("%s/doc/", baseUrl),
	})

	// get a list of all collections
	pool := database.GetInstance(ctx)
	rows, err := pool.Query(ctx, "SELECT id, content->>'title'::text as title FROM pgstac.collections ORDER BY id")
	if err != nil {
		log.Error().Err(err).Msg("error querying collections for catalog response")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Error{
			Code:        database.QueryErrorCode,
			Description: "could not query collections table",
		})
	}
	defer rows.Close()
	for rows.Next() {
		child := stac.Link{
			Rel:  "child",
			Type: "application/json",
		}
		var collectionId string
		err := rows.Scan(&collectionId, &child.Title)
		if err != nil {
			log.Error().Err(err).Msg("could not scan collection id and title")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        database.QueryErrorCode,
				Description: "could not serialize data from collections table",
			})
		}
		child.Href = fmt.Sprintf("%s/collections/%s", self, collectionId)
		links = append(links, child)
	}

	catalog := stac.Catalog{
		Type:        "Catalog",
		ID:          viper.GetString("stac.catalog.id"),
		Title:       viper.GetString("stac.catalog.title"),
		Description: viper.GetString("stac.catalog.description"),
		StacVersion: "1.0.0",
		ConformsTo:  stac.Conformance,
		Links:       links,
	}

	return c.JSON(catalog)
}

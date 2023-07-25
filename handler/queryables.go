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

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/stac"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Queryables(c *fiber.Ctx) error {
	ctx := context.Background()
	collectionId := c.Params("collectionId")
	var raw json.RawMessage

	pCollectionId := &collectionId
	if collectionId == "" {
		pCollectionId = nil
	}

	pool := database.GetInstance(ctx)
	if err := pool.QueryRow(ctx, "SELECT get_queryables FROM get_queryables($1::text)", pCollectionId).Scan(&raw); err != nil {
		log.Error().Err(err).Str("collection", collectionId).Msg("failed to get queryables from database")
		c.Status(fiber.ErrInternalServerError.Code)
		return c.JSON(stac.Message{
			Code:        "DatabaseError",
			Description: "failed to get queryables from database",
		})
	}

	return c.JSON(raw)
}

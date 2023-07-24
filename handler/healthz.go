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
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Collection returns details of a specific collection
func Healthz(c *fiber.Ctx) error {
	ctx := context.Background()

	overallHealth := "OK"
	dbHealth := "OK"

	pool := database.GetInstance(ctx)
	if err := pool.Ping(ctx); err != nil {
		log.Error().Err(err).Msg("database ping failed")
		dbHealth = "FAILED"
		overallHealth = "FAILED"
	}

	return c.JSON(map[string]string{
		"status":   overallHealth,
		"database": dbHealth,
	})
}

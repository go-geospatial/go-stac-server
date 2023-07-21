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

package router

import (
	"github.com/go-geospatial/go-stac-server/handler"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes setup router api
func SetupRoutes(app *fiber.App) {
	// config.js
	app.Get("/config.js", handler.StacBrowserConfig)
	// STAC API
	api := app.Group("api")
	stac := api.Group("stac")
	stacV1 := stac.Group("v1")

	stacV1.Get("/", handler.Catalog)
	stacV1.Get("/collections", handler.Collections)
	stacV1.Get("/conformance", handler.Conformance)
	stacV1.Get("/collections/:id", handler.Collection)
	stacV1.Get("/collections/:id/items", handler.Items)
	stacV1.Get("/collections/:collectionId/items/:itemId", handler.Item)
}

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
	// config.js - used to configure Stac Browser
	app.Get("/config.js", handler.StacBrowserConfig)

	// healthz
	app.Get("/healthz", handler.Healthz)
	// NOTE: prometheus also registers a /metrics endpoint

	// STAC API
	api := app.Group("api")
	stac := api.Group("stac")
	stacV1 := stac.Group("v1")

	stacV1.Get("/", handler.Catalog)
	stacV1.Get("/collections", handler.Collections)
	stacV1.Get("/conformance", handler.Conformance)
	stacV1.Get("/collections/:collectionId", handler.Collection)
	stacV1.Get("/collections/:collectionId/items", handler.Items)
	stacV1.Get("/collections/:collectionId/items/:itemId", handler.Item)

	stacV1.Get("/search", handler.Search)
	stacV1.Post("/search", handler.Search)

	// Filter Extension
	stacV1.Get("/collections/:collectionId/queryables", handler.Queryables)
	stacV1.Get("/queryables", handler.Queryables)

	// Transactions extension
	stacV1.Post("/collections", handler.ModifyCollection)
	stacV1.Put("/collections", handler.ModifyCollection)
	stacV1.Delete("/collections/:collectionId", handler.DeleteCollection)

	stacV1.Post("/collections/:collectionId/items", handler.CreateItems)
	//stacV1.Delete("/collections/:collectionId/items/:itemId", handler.DeleteItem)
	//stacV1.Put("/collections/:collectionId/items/:itemId", handler.UpdateItem)
	//stacV1.Patch("/collections/:collectionId/items/:itemId", handler.UpdateItem)
}

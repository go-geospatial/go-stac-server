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

package static

import (
	"embed"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/rs/zerolog/log"
)

//go:embed files/openapi.json
var openAPI string

//go:embed files/*
var f embed.FS

func OpenAPIHandler(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/vnd.oai.openapi+json;version=3.1")
	return c.SendString(openAPI)
}

func InitStaticFiles(app *fiber.App) {
	log.Info().Msg("seting up filesystem")

	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(f),
		PathPrefix: "files",
		Browse:     true,
	}))
}

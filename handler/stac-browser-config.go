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

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func StacBrowserConfig(c *fiber.Ctx) error {
	var data []byte
	if viper.GetString("gui.config") != "" {
		data = []byte(viper.GetString("gui.config"))
	} else {
		data = []byte(fmt.Sprintf("window.STAC_BROWSER_CONFIG = {catalogUrl: \"%s/api/stac/v1\"}", getBaseURL(c)))
	}
	_, err := c.Status(200).Write(data)
	return err
}

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

package common

import (
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// JSON converts any interface or string to JSON.
// Array and slice values encode as JSON arrays,
// except that []byte encodes as a base64-encoded string,
// and a nil slice encodes as the null JSON value.
// This method also sets the content header to application/json.
func GeoJSON(c *fiber.Ctx, data interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.Set("Content-Type", "application/geo+json")
	return c.Send(raw)
}

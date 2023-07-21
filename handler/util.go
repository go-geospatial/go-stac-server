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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-geospatial/go-stac-server/stac"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func getBaseUrl(c *fiber.Ctx) string {
	return fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname())
}

func parseLimit(c *fiber.Ctx, limitStr string) (int, error) {
	var err error
	var limit int
	if limit, err = strconv.Atoi(limitStr); err != nil {
		log.Error().Err(err).Str("limit", limitStr).Msg("could not convert limit to int")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return 0, c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: fmt.Sprintf("limit '%s' could not be converted to int", limitStr),
		})
	}
	if limit < 0 || limit > 1000 {
		log.Error().Err(err).Int("limit", limit).Msg("limit out of bounds: 1 <= limit <= 1000")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return 0, c.JSON(stac.Error{
			Code:        stac.ParameterError,
			Description: fmt.Sprintf("limit '%s' must be between 1 and 1000", limitStr),
		})
	}

	return limit, nil
}

func parseRFC3339Date(c *fiber.Ctx, dateStr string) error {
	if dateStr != "" {
		if matched, err := regexp.MatchString(`(20\d{2})-(0[1-9]|1[0-2])-([01][\d]|3[01])[\sT]([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(Z|[\+-](0[\d]|1[\d]|2[0-3]):([0-5]\d))?`, dateStr); err != nil {
			log.Error().Err(err).Msg("could not validate date str due to regex error")
			c.Status(fiber.ErrInternalServerError.Code)
			return c.JSON(stac.Error{
				Code:        stac.ServerError,
				Description: fmt.Sprintf("regex error while validating datetime"),
			})
		} else if !matched {
			c.Status(fiber.ErrUnprocessableEntity.Code)
			return c.JSON(stac.Error{
				Code:        stac.ParameterError,
				Description: fmt.Sprintf("datetime '%s' is not RFC 3339 formated", dateStr),
			})
		}
	}

	return nil
}

func parseBbox(c *fiber.Ctx, bboxStr string) ([]float64, error) {
	bbox := make([]float64, 4)
	if bboxStr != "" {
		if err := json.Unmarshal([]byte(bboxStr), &bbox); err != nil {
			log.Error().Err(err).Msg("could not validate date str due to regex error")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			return nil, c.JSON(stac.Error{
				Code:        stac.ParameterError,
				Description: fmt.Sprintf("could not parse bbox JSON: '%s'; must be of the form [num, num, num, num]", bboxStr),
			})
		}
	}

	return bbox, nil
}

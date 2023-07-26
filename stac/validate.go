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

package stac

import (
	"errors"
	"regexp"

	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func ValidateCollectionIDsMatch(c *fiber.Ctx, obj map[string]*json.RawMessage, expected string) error {
	if specifiedCollectionID, ok := obj["collection"]; ok {
		var specified string
		if err := json.Unmarshal(*specifiedCollectionID, &specified); err != nil {
			log.Error().Err(err).Msg("invalid items json - collections parameter unmarshable")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			err2 := c.JSON(Message{
				Code:        ParameterError,
				Description: "invalid items json - collections parameter unmarshable",
			})
			if err2 != nil {
				return err
			}
			return err
		}

		if specified != expected {
			err := errors.New("collection parameters do not match")
			log.Error().Str("URL-parameter", expected).Str("JSON-parameter", specified).Msg("collection parameters do not match")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			_ = c.JSON(Message{
				Code:        ParameterError,
				Description: "collection path id does not match json collection id",
			})
			return err
		}
	} else {
		err := errors.New("invalid items json - collections parameter missing")
		log.Error().Msg("invalid items json - collections parameter missing")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		_ = c.JSON(Message{
			Code:        ParameterError,
			Description: "invalid items json - collections parameter missing",
		})
		return err
	}

	return nil
}

func ValidateID(c *fiber.Ctx, obj map[string]*json.RawMessage) (string, error) {
	// validate the ID field
	idRe := regexp.MustCompile(`^([a-zA-Z0-9\-_\.]+)$`)
	var id string
	if idRaw, ok := obj["id"]; ok {
		if err := json.Unmarshal(*idRaw, &id); err != nil {
			log.Error().Err(err).Str("id", string(*idRaw)).Msg("cannot parse id string")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			_ = c.JSON(Message{
				Code:        ParameterError,
				Description: `cannot parse id string must conform to format: '^([a-zA-Z0-9\-_\.]+)$'`,
			})
			return "", err
		}
	} else {
		err := errors.New("id field is missing from create collection")
		log.Error().Msg("id field is missing from create collection")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		_ = c.JSON(Message{
			Code:        ParameterError,
			Description: `id field is required`,
		})
		return "", err
	}

	if !idRe.MatchString(id) {
		err := errors.New("id field contains invalid characters")
		log.Error().Msg("id field contains invalid characters")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		_ = c.JSON(Message{
			Code:        ParameterError,
			Description: `id must conform to format '^([a-zA-Z0-9\-_\.]+)$'`,
		})
		return "", err
	}

	return id, nil
}

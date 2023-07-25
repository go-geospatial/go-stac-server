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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-geospatial/go-stac-server/stac"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func buildQueryArray(c *fiber.Ctx) []string {
	queryParts := make([]string, 0, 5)
	possible := []string{"collections", "limit", "bbox", "datetime", "filter", "filter-lang", "sortby", "fields"}

	for _, key := range possible {
		val := c.Query(key, "")
		if val != "" {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", key, val))
		}
	}

	return queryParts
}

func getCQLFromBody(c *fiber.Ctx) (stac.CQL, error) {
	var cql stac.CQL
	if err := json.Unmarshal(c.Body(), &cql); err != nil {
		log.Error().Err(err).Msg("could not parse search body")
		c.Status(fiber.ErrInternalServerError.Code)
		return stac.CQL{}, c.JSON(stac.Message{
			Code:        stac.ServerError,
			Description: "could not parse search body",
		})
	}

	// update default value for limit
	if cql.Limit == 0 {
		cql.Limit = 10
	}

	if err := validateLimit(c, cql.Limit); err != nil {
		return stac.CQL{}, err
	}

	if cql.FilterLang == "" {
		cql.FilterLang = "cql-json"
	}

	return cql, nil
}

func getCQLFromQuery(c *fiber.Ctx) (stac.CQL, error) {
	collectionsStr := c.Query("collections", "")
	limitStr := c.Query("limit", "10")
	bboxStr := c.Query("bbox", "")
	dateStr := c.Query("datetime", "")
	filterStr := c.Query("filter", "")
	filterLang := c.Query("filter-lang", "cql2-text")
	sortByStr := c.Query("sortby", "")
	fieldStr := c.Query("fields", "")
	token := c.Query("token", "")

	// parse collections
	collections, err := parseCollections(c, collectionsStr)
	if err != nil {
		// response and logging handled by parseCollections
		return stac.CQL{}, err
	}

	// parse limit
	limit, err := parseLimit(c, limitStr)
	if err != nil {
		// response and logging handled by parseLimit
		return stac.CQL{}, err
	}

	// parse bbox
	bbox, err := parseBboxQuery(c, bboxStr)
	if err != nil {
		// response and logging handled by parseBbox
		return stac.CQL{}, err
	}

	// parse date string (must be RFC 3339)
	if err := parseRFC3339Date(c, dateStr); err != nil {
		// http response and logging handled by parseRFC3339Date
		return stac.CQL{}, err
	}

	// parse CQL-2 filter
	var filter *json.RawMessage
	if filter, filterLang, err = parseCQL2Filter(c, filterStr, filterLang); err != nil {
		// http response and logging handled by parseCQL2Filter
		return stac.CQL{}, err
	}

	// parse sortby
	var sort []stac.CQLSort
	if sort, err = parseSort(c, sortByStr); err != nil {
		// http response and logging handled by parseSort
		return stac.CQL{}, err
	}

	// parse fields
	var fields stac.CQLFields
	if fields, err = parseFields(c, fieldStr); err != nil {
		// http response and logging handled by parseFields
		return stac.CQL{}, err
	}

	// create CQL search criteria
	conf := json.RawMessage(`{"nohydrate": false}`)
	cql := stac.CQL{
		Limit:    limit,
		DateTime: dateStr,
		Token:    token,
		Conf:     &conf,
	}

	if collectionsStr != "" {
		cql.Collections = collections
	}

	if fieldStr != "" {
		cql.Fields = &fields
	}

	if bboxStr != "" {
		cql.Bbox = bbox
	}

	if filterStr != "" {
		cql.Filter = filter
		cql.FilterLang = filterLang
	} else {
		cql.FilterLang = "cql-json"
	}

	if sortByStr != "" {
		cql.SortBy = sort
	}

	return cql, nil
}

func parseCollections(c *fiber.Ctx, collectionsStr string) ([]string, error) {
	var collections []string
	if collectionsStr != "" {
		collections = strings.Split(collectionsStr, ",")
	}
	return collections, nil
}

func parseSort(c *fiber.Ctx, sortByStr string) ([]stac.CQLSort, error) {
	var sort []stac.CQLSort
	if sortByStr != "" {
		sort = make([]stac.CQLSort, 0, 1)
		sortRe, err := regexp.Compile(`^([\+-]?)(.*)$`)
		if err != nil {
			log.Error().Err(err).Msg("could not compile sortby regular expression")
			c.Status(fiber.ErrInternalServerError.Code)
			c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "regex error while validating sortby",
			})
			return sort, err
		}
		tokens := strings.Split(sortByStr, ",")
		for _, token := range tokens {
			groups := sortRe.FindStringSubmatch(token)
			if len(groups) > 0 {
				direction := "asc"
				if groups[1] == "-" {
					direction = "desc"
				}
				sort = append(sort, stac.CQLSort{
					Field:     groups[2],
					Direction: direction,
				})
			} else {
				err := errors.New("sort field does not match regex")
				log.Error().Err(err).Msg("sort field does not match regex")
				c.Status(fiber.ErrInternalServerError.Code)
				c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "sort expression must be of the form ([+-]?)(.*)",
				})
				return sort, err
			}
		}
	}

	return sort, nil
}

func parseFields(c *fiber.Ctx, fieldStr string) (stac.CQLFields, error) {
	var fields stac.CQLFields
	if fieldStr != "" {
		fields.Include = make([]string, 0, 5)
		fields.Exclude = make([]string, 0, 5)

		fieldsRe, err := regexp.Compile(`^([\+-]?)(.*)$`)
		if err != nil {
			log.Error().Err(err).Msg("could not compile fields regular expression")
			c.Status(fiber.ErrInternalServerError.Code)
			c.JSON(stac.Message{
				Code:        stac.ServerError,
				Description: "regex error while validating fields",
			})
			return fields, err
		}

		tokens := strings.Split(fieldStr, ",")
		for _, token := range tokens {
			groups := fieldsRe.FindStringSubmatch(token)
			if len(groups) > 0 {
				if groups[1] == "-" {
					fields.Exclude = append(fields.Exclude, groups[2])
				} else {
					fields.Include = append(fields.Include, groups[2])
				}
			} else {
				err := errors.New("sort field does not match regex")
				log.Error().Err(err).Msg("sort field does not match regex")
				c.Status(fiber.ErrInternalServerError.Code)
				c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "fields must be of the form ([-]?)(.*)",
				})
				return fields, err
			}
		}
	}

	return fields, nil
}

func parseLimit(c *fiber.Ctx, limitStr string) (int, error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		log.Error().Err(err).Str("limit", limitStr).Msg("could not convert limit to int")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: fmt.Sprintf("limit '%s' could not be converted to int", limitStr),
		})
		return 0, err
	}

	if err := validateLimit(c, limit); err != nil {
		return 0, err
	}

	return limit, nil
}

func validateLimit(c *fiber.Ctx, limit int) error {
	if limit < 0 || limit > 10000 {
		err := errors.New("limit out of bounds")
		log.Error().Err(err).Int("limit", limit).Msg("limit out of bounds: 1 <= limit <= 10,000")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: fmt.Sprintf("limit '%d' must be between 1 and 10,000", limit),
		})
		return err
	}
	return nil
}

func parseRFC3339Date(c *fiber.Ctx, dateStr string) error {
	if dateStr != "" {
		if first, second, found := strings.Cut(dateStr, "/"); found {
			// interval specified
			re, err := regexp.Compile(`(\d{4})-(0[1-9]|1[0-2])-([01][\d]|3[01])[\sT]([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(Z|[\+-](0[\d]|1[\d]|2[0-3]):([0-5]\d))?|\.\.`)
			if err != nil {
				log.Error().Err(err).Msg("could not compile RFC3339 regular expression")
				c.Status(fiber.ErrInternalServerError.Code)
				return c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "regex error while validating datetime",
				})
			}

			if first == ".." && second == ".." {
				// both parts of the interval cannot be open
				c.Status(fiber.ErrUnprocessableEntity.Code)
				return c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("both sides of the interval cannot be open", dateStr),
				})
			}

			if matched := re.MatchString(first); !matched {
				c.Status(fiber.ErrUnprocessableEntity.Code)
				return c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("first datetime '%s' is not RFC 3339 formated or open", first),
				})
			}

			if matched := re.MatchString(second); !matched {
				c.Status(fiber.ErrUnprocessableEntity.Code)
				return c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("second datetime '%s' is not RFC 3339 formated or open", second),
				})
			}

		} else {
			// single date specified
			if matched, err := regexp.MatchString(`(\d{4})-(0[1-9]|1[0-2])-([01][\d]|3[01])[\sT]([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(Z|[\+-](0[\d]|1[\d]|2[0-3]):([0-5]\d))?`, first); err != nil {
				log.Error().Err(err).Msg("could not validate date str due to regex error")
				c.Status(fiber.ErrInternalServerError.Code)
				return c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "regex error while validating datetime",
				})
			} else if !matched {
				c.Status(fiber.ErrUnprocessableEntity.Code)
				return c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("datetime '%s' is not RFC 3339 formated", dateStr),
				})
			}
		}
	}

	return nil
}

func parseBboxQuery(c *fiber.Ctx, bboxStr string) ([]float64, error) {
	bbox := make([]float64, 0, 6)
	if bboxStr != "" {
		bboxParts := strings.Split(bboxStr, ",")
		for _, bboxCoord := range bboxParts {
			if coord, err := strconv.ParseFloat(bboxCoord, 64); err != nil {
				log.Error().Err(err).Str("Coord", bboxCoord).Msg("could not convert bbox coordinate to float64")
				c.Status(fiber.ErrUnprocessableEntity.Code)
				return nil, c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("could not parse bbox: '%s'; offending coordinate '%s'. bbox must be 4 or 6 float64 coordinates separated by commas. The coordinate order is: lower left axis-1, lower left axis-2, minimum axis-3 (optional), upper right axis-1, upper right axis-2, maximum axis-3 (optional)", bboxStr, bboxCoord),
				})
			} else {
				bbox = append(bbox, coord)
			}
		}

		if len(bbox) != 4 || len(bbox) != 6 {
			err := errors.New("bbox must be length 4 or 6")
			log.Error().Err(err).Str("bbox", bboxStr).Msg("bbox parse error")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			return nil, c.JSON(stac.Message{
				Code:        stac.ParameterError,
				Description: fmt.Sprintf("could not parse bbox: '%s'; bbox must be 4 or 6 float64 coordinates separated by commas. The coordinate order is: lower left axis-1, lower left axis-2, minimum axis-3 (optional), upper right axis-1, upper right axis-2, maximum axis-3 (optional)", bboxStr),
			})
		}
	}

	return bbox, nil
}

func parseStringListQuery(c *fiber.Ctx, queryStr string) ([]string, error) {
	var list []string
	if queryStr != "" {
		list = strings.Split(queryStr, ",")
	}
	return list, nil
}

func parseIntersectsQuery(c *fiber.Ctx, intersectsStr string) (*stac.GeoJson, error) {
	var intersects stac.GeoJson
	if intersectsStr != "" {
		if err := json.Unmarshal([]byte(intersectsStr), &intersects); err != nil {
			log.Error().Err(err).Str("intersects", intersectsStr).Msg("error parsing GeoJson intersects query")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			return nil, c.JSON(stac.Message{
				Code:        stac.ParameterError,
				Description: "could not parse intersects query",
			})
		}
	}
	return &intersects, nil
}

func parseCQL2Filter(c *fiber.Ctx, filterStr string, filterLang string) (*json.RawMessage, string, error) {
	var jsonRaw json.RawMessage

	// validate CQL2 text
	switch filterLang {
	case "cql2-text":
		// convert to cql2-json
		jsonRaw = []byte(filterStr)
	case "cql2-json":
		// validate cql2-json against json-schema
		jsonRaw = []byte(filterStr)
	default:
		err := errors.New("filter-lang must be one of 'cql2-text' or 'cql2-json'")
		log.Error().Err(err).Str("filter-lang", filterLang).Msg("invalid filter-lang provided")
		c.Status(fiber.ErrUnprocessableEntity.Code)
		return nil, "cql-json", c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "invalid filter-lang provided",
		})
	}

	return &jsonRaw, "cql2-json", nil
}

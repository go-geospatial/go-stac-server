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

var CQLJSON = "cql-json"
var CQL2JSON = "cql2-json"
var CQLText = "cql2-text"

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
	fmt.Println(string(c.Body()))
	if err := json.Unmarshal(c.Body(), &cql); err != nil {
		log.Error().Err(err).Msg("could not parse search body")
		c.Status(fiber.StatusBadRequest)
		return stac.CQL{}, c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "could not parse search body",
		})
	}

	if len(cql.Bbox) != 0 && cql.Intersects != nil {
		log.Error().Msg("cannot specify both bbox and intersects")
		c.Status(fiber.StatusBadRequest)
		c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "cannot specify both bbox and intersects",
		})
		return stac.CQL{}, errors.New("cannot specify both bbox and intersects")
	}

	// update default value for limit
	if cql.Limit == 0 {
		cql.Limit = 10
	}

	if limit, err := validateLimit(c, cql.Limit); err == nil {
		cql.Limit = limit
	} else {
		return stac.CQL{}, err
	}

	// validate bbox
	if _, err := validateBbox(c, cql.Bbox); err != nil {
		return stac.CQL{}, err
	}

	if cql.FilterLang == "" {
		cql.FilterLang = CQLJSON
	}

	return cql, nil
}

func getCQLFromQuery(c *fiber.Ctx) (stac.CQL, error) {
	collectionsStr := c.Query("collections", "")
	limitStr := c.Query("limit", "10")
	intersectsStr := c.Query("intersects", "")
	bboxStr := c.Query("bbox", "")
	dateStr := c.Query("datetime", "")
	filterStr := c.Query("filter", "")
	filterLang := c.Query("filter-lang", "cql2-text")
	sortByStr := c.Query("sortby", "")
	fieldStr := c.Query("fields", "")
	token := c.Query("token", "")

	if bboxStr != "" && intersectsStr != "" {
		log.Error().Msg("cannot specify both bbox and intersects")
		c.Status(fiber.StatusBadRequest)
		c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "cannot specify both bbox and intersects",
		})
		return stac.CQL{}, errors.New("cannot specify both bbox and intersects")
	}

	// parse collections
	collections := parseCollections(collectionsStr)

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

	// parse intersects
	var intersects *stac.GeoJSON
	if intersects, err = parseIntersects(c, intersectsStr); err != nil {
		// http response and logging handled by parseIntersectsQuery
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

	if intersectsStr != "" {
		cql.Intersects = intersects
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

func parseCollections(collectionsStr string) []string {
	var collections []string
	if collectionsStr != "" {
		collections = strings.Split(collectionsStr, ",")
	}
	return collections
}

func parseSort(c *fiber.Ctx, sortByStr string) ([]stac.CQLSort, error) {
	var sort []stac.CQLSort
	if sortByStr != "" {
		sort = make([]stac.CQLSort, 0, 1)
		sortRe := regexp.MustCompile(`^([\+-]?)(.*)$`)
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
				_ = c.JSON(stac.Message{
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

		fieldsRe := regexp.MustCompile(`^([\+-]?)(.*)$`)
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
				_ = c.JSON(stac.Message{
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
		_ = c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: fmt.Sprintf("limit '%s' could not be converted to int", limitStr),
		})
		return 0, err
	}

	return validateLimit(c, limit)
}

func validateLimit(c *fiber.Ctx, limit int) (int, error) {
	if limit > 10_000 {
		log.Warn().Int("limit", limit).Msg("limit out of bounds: limit > 10,000")
		return 10_000, nil
	}
	if limit < 0 {
		err := errors.New("limit out of bounds")
		log.Warn().Int("limit", limit).Msg("limit out of bounds: limit < 0")
		c.Status(fiber.StatusBadRequest)
		_ = c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: fmt.Sprintf("limit '%d' must be greater than 0", limit),
		})
		return 0, err
	}
	return limit, nil
}

func parseRFC3339Date(c *fiber.Ctx, dateStr string) error {
	if dateStr != "" {
		if first, second, found := strings.Cut(dateStr, "/"); found {
			// interval specified
			re := regexp.MustCompile(`(\d{4})-(0[1-9]|1[0-2])-([01][\d]|3[01])[\sT]([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(Z|[\+-](0[\d]|1[\d]|2[0-3]):([0-5]\d))?|\.\.`)
			if first == ".." && second == ".." {
				// both parts of the interval cannot be open
				c.Status(fiber.ErrUnprocessableEntity.Code)
				_ = c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("both sides of the interval cannot be open: %s", dateStr),
				})
				return errors.New("both parts of the interval cannot be open")
			}

			if matched := re.MatchString(first); !matched {
				c.Status(fiber.ErrUnprocessableEntity.Code)
				_ = c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("first datetime '%s' is not RFC 3339 formatted or open", first),
				})
				return errors.New("datetime is not RFC 3339 formatted")
			}

			if matched := re.MatchString(second); !matched {
				c.Status(fiber.ErrUnprocessableEntity.Code)
				c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("second datetime '%s' is not RFC 3339 formated or open", second),
				})
				return errors.New("datetime is not RFC 3339 formatted")
			}

		} else {
			// single date specified
			if matched, err := regexp.MatchString(`(\d{4})-(0[1-9]|1[0-2])-([01][\d]|3[01])[\sT]([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(Z|[\+-](0[\d]|1[\d]|2[0-3]):([0-5]\d))?`, first); err != nil {
				log.Error().Err(err).Msg("could not validate date str due to regex error")
				c.Status(fiber.ErrInternalServerError.Code)
				_ = c.JSON(stac.Message{
					Code:        stac.ServerError,
					Description: "regex error while validating datetime",
				})
				return err
			} else if !matched {
				c.Status(fiber.ErrUnprocessableEntity.Code)
				_ = c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("datetime '%s' is not RFC 3339 formated", dateStr),
				})
				return errors.New("datetime is not RFC 3339")
			}
		}
	}

	return nil
}

func parseBboxQuery(c *fiber.Ctx, bboxStr string) ([]float64, error) {
	var err error
	bbox := make([]float64, 0, 6)
	if bboxStr != "" {
		bboxParts := strings.Split(bboxStr, ",")
		for _, bboxCoord := range bboxParts {
			var coord float64
			if coord, err = strconv.ParseFloat(bboxCoord, 64); err != nil {
				log.Error().Err(err).Str("Coord", bboxCoord).Msg("could not convert bbox coordinate to float64")
				c.Status(fiber.StatusBadRequest)
				_ = c.JSON(stac.Message{
					Code:        stac.ParameterError,
					Description: fmt.Sprintf("could not parse bbox: '%s'; offending coordinate '%s'. bbox must be 4 or 6 float64 coordinates separated by commas. The coordinate order is: lower left axis-1, lower left axis-2, minimum axis-3 (optional), upper right axis-1, upper right axis-2, maximum axis-3 (optional)", bboxStr, bboxCoord),
				})
				return nil, err
			}
			bbox = append(bbox, coord)
		}
	}

	return validateBbox(c, bbox)
}

func validateBbox(c *fiber.Ctx, bbox []float64) ([]float64, error) {
	if len(bbox) != 0 || len(bbox) != 4 || len(bbox) != 6 {
		err := errors.New("bbox must be length 4 or 6")
		log.Error().Err(err).Floats64("bbox", bbox).Msg("bbox ivalid length. must be 4 or 6.")
		c.Status(fiber.StatusBadRequest)
		_ = c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "bbox of invalid length",
		})
		return nil, err
	}

	if len(bbox) == 4 && bbox[1] > bbox[3] {
		err := errors.New("bbox lat1 > lat2")
		log.Error().Err(err).Floats64("bbox", bbox).Msg("lat1 > lat2")
		c.Status(fiber.StatusBadRequest)
		_ = c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "bbox invalid lat1 > lat2",
		})
		return nil, err
	}

	if len(bbox) == 6 && bbox[1] > bbox[4] {
		err := errors.New("bbox lat1 > lat2")
		log.Error().Err(err).Floats64("bbox", bbox).Msg("lat1 > lat2")
		c.Status(fiber.StatusBadRequest)
		_ = c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "bbox invalid lat1 > lat2",
		})
		return nil, err
	}

	return bbox, nil
}

func parseIntersects(c *fiber.Ctx, intersectsStr string) (*stac.GeoJSON, error) {
	var intersects stac.GeoJSON
	if intersectsStr != "" {
		if err := json.Unmarshal([]byte(intersectsStr), &intersects); err != nil {
			log.Error().Err(err).Str("intersects", intersectsStr).Msg("error parsing GeoJson intersects query")
			c.Status(fiber.ErrUnprocessableEntity.Code)
			_ = c.JSON(stac.Message{
				Code:        stac.ParameterError,
				Description: "could not parse intersects query",
			})
			return nil, err
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
		_ = c.JSON(stac.Message{
			Code:        stac.ParameterError,
			Description: "invalid filter-lang provided",
		})
		return nil, "cql-json", err
	}

	return &jsonRaw, "cql2-json", nil
}

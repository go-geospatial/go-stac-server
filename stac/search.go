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
	"context"

	"github.com/go-geospatial/go-stac-server/database"
	json "github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

type CQLSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type CQLFields struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type CQL struct {
	Collections []string         `json:"collections,omitempty"`
	Ids         []string         `json:"ids,omitempty"`
	Bbox        []float64        `json:"bbox,omitempty"`
	Intersects  *GeoJSON         `json:"intersects,omitempty"`
	DateTime    string           `json:"datetime,omitempty"`
	Limit       int              `json:"limit"`
	Conf        *json.RawMessage `json:"conf,omitempty"`
	Query       *json.RawMessage `json:"query,omitempty"`
	Fields      *CQLFields       `json:"fields,omitempty"`
	SortBy      []CQLSort        `json:"sortby,omitempty"`
	Filter      *json.RawMessage `json:"filter,omitempty"`
	FilterLang  string           `json:"filter-lang"`
	Token       string           `json:"token,omitempty"`
}

type SearchResponse struct {
	Context  *json.RawMessage              `json:"context"`
	Type     string                        `json:"string"`
	Features []map[string]*json.RawMessage `json:"features"`
	Next     string                        `json:"next"`
	Prev     string                        `json:"prev"`
}

// Item returns details of a specific item
func Search(params CQL) (*SearchResponse, error) {
	ctx := context.Background()

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal search parameters")
		return nil, err
	}

	pool := database.GetInstance(ctx)
	row := pool.QueryRow(ctx, "SELECT search FROM search($1::text::jsonb)", paramsJSON)

	var searchJSON []byte
	if err := row.Scan(&searchJSON); err != nil {
		log.Error().Err(err).Msg("failed to scan JSON from postgresql search query")
		return nil, err
	}

	var searchResponse SearchResponse
	if err = json.Unmarshal(searchJSON, &searchResponse); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal search JSON")
		return nil, err
	}

	return &searchResponse, nil
}

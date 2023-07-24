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

type Catalog struct {
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	StacVersion string   `json:"stac_version"`
	ConformsTo  []string `json:"conformsTo"`
	Links       []Link   `json:"links"`
}

var Conformance []string = []string{
	"http://www.opengis.net/spec/cql2/1.0/conf/basic-cql2",
	"http://www.opengis.net/spec/cql2/1.0/conf/cql2-json",
	// TODO: "http://www.opengis.net/spec/cql2/1.0/conf/cql2-text",
	"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/core",
	"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/geojson",
	"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/oas30",
	"http://www.opengis.net/spec/ogcapi-features-3/1.0/conf/filter",
	"http://www.opengis.net/spec/ogcapi-features-3/1.0/conf/features-filter",
	"https://api.stacspec.org/v1.0.0/collections",
	"https://api.stacspec.org/v1.0.0/core",
	"https://api.stacspec.org/v1.0.0-rc.3/browseable",
	"https://api.stacspec.org/v1.0.0/item-search",
	"https://api.stacspec.org/v1.0.0-rc.2/item-search#context",
	"https://api.stacspec.org/v1.0.0-rc.3/item-search#fields",
	"https://api.stacspec.org/v1.0.0-rc.2/item-search#filter",
	"https://api.stacspec.org/v1.0.0-rc.2/item-search#query",
	"https://api.stacspec.org/v1.0.0-rc.2/item-search#sort",
	"https://api.stacspec.org/v1.0.0/ogcapi-features",
	"https://api.stacspec.org/v1.0.0-rc.3/ogcapi-features#fields",
	"https://api.stacspec.org/v1.0.0-rc.2/ogcapi-features#sort",
	"https://api.stacspec.org/v1.0.0-rc.2/ogcapi-features/extensions/transaction",
	"http://www.opengis.net/spec/ogcapi-features-4/1.0/conf/simpletx",
}

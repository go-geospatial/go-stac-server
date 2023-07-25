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
	"fmt"

	json "github.com/goccy/go-json"
)

type Link struct {
	Rel    string           `json:"rel"`
	Type   string           `json:"type"`
	Title  string           `json:"title,omitempty"`
	Href   string           `json:"href"`
	Method string           `json:"method,omitempty"`
	Body   *json.RawMessage `json:"body,omitempty"`
}

// AddLink creates a new link reference in the Links array of a Feature
// rel is the name of the link relationship
// baseUrl baseUrl of this STAC server
// endpoint is the last portion of the URL i.e. <base url>/api/stac/v1/<endpoint>
func AddLink(links []Link, baseURL string, rel string, endpoint string, mimeType string) []Link {
	href := fmt.Sprintf("%s/api/stac/v1%s", baseURL, endpoint)
	links = append(links, Link{
		Rel:  rel,
		Type: mimeType,
		Href: href,
	})

	return links
}

// AddLinkPost creates a new link reference in the Links array of a Feature
// rel is the name of the link relationship
// baseUrl baseUrl of this STAC server
// endpoint is the last portion of the URL i.e. <base url>/api/stac/v1/<endpoint>
func AddLinkPost(links []Link, baseURL string, rel string, endpoint string, mimeType string, body *json.RawMessage) []Link {
	href := fmt.Sprintf("%s/api/stac/v1%s", baseURL, endpoint)
	links = append(links, Link{
		Rel:    rel,
		Type:   mimeType,
		Href:   href,
		Method: "POST",
		Body:   body,
	})

	return links
}

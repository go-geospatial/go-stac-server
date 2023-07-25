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

package jsonutil

import (
	json "github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

func isObject(a []byte) bool {
	return a[0] == '{'
}

// Merge takes 2 JSON strings and recursively merges a into b
func Merge(a, b []byte) (json.RawMessage, error) {
	aMap := make(map[string]*json.RawMessage)
	bMap := make(map[string]*json.RawMessage)

	if err := json.Unmarshal(a, &aMap); err != nil {
		log.Error().Err(err).Str("a", string(a)).Msg("cannot unmarshal JSON")
		return []byte{}, err
	}

	if err := json.Unmarshal(b, &bMap); err != nil {
		log.Error().Err(err).Str("b", string(b)).Msg("cannot unmarshal JSON")
		return []byte{}, err
	}

	for k, aFragment := range aMap {
		if bFragment, ok := bMap[k]; ok {
			if isObject(*aFragment) && isObject(*bFragment) {
				merged, err := Merge(*aFragment, *bFragment)
				if err != nil {
					log.Error().Err(err).Msg("cannot merge JSON")
					return []byte{}, err
				}
				bMap[k] = &merged
			} else {
				bMap[k] = aFragment
			}
		} else {
			bMap[k] = aFragment
		}
	}

	if result, err := json.Marshal(bMap); err != nil {
		log.Error().Err(err).Msg("cannot marshal b JSON")
		return []byte{}, err
	} else {
		return result, nil
	}
}

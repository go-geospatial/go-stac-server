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

package database

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	once     sync.Once
	instance *pgxpool.Pool
)

var ConnectionErrorCode = "DatabaseConnectionError"
var ConnectionErrorDescription = "Could not connect to database"
var QueryErrorCode = "DatabaseQueryError"

func GetInstance(ctx context.Context) *pgxpool.Pool {
	once.Do(func() {
		// mask DSN password for logging
		re := regexp.MustCompile(`^postgresql://(\w+)(:(password))?(.*)`)
		dsnMasked := viper.GetString("database.dsn")
		parts := re.FindStringSubmatch(viper.GetString("database.dsn"))
		if len(parts) == 0 {
			log.Warn().Msg("DSN format not recognized, password masking disabled")
		} else {
			dsnMasked = fmt.Sprintf("postgresql://%s:***%s", parts[1], parts[4])
		}

		log.Info().Str("DSN", dsnMasked).Msg("initializing database pool connection")
		var err error
		instance, err = pgxpool.New(ctx, viper.GetString("database.dsn"))
		if err != nil {
			log.Error().Err(err).Msg("failed to create a new database pool")
			os.Exit(66)
		}
	})
	return instance
}

func Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	dbpool := GetInstance(ctx)
	return dbpool.Acquire(ctx)
}

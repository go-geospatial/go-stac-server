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
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	once     sync.Once
	instance *pgxpool.Pool
)

var ConnectionErrorCode string = "DatabaseConnectionError"
var ConnectionErrorDescription string = "Could not connect to database"
var QueryErrorCode string = "DatabaseQueryError"

func GetInstance(ctx context.Context) *pgxpool.Pool {
	once.Do(func() {
		log.Debug().Str("DSN", viper.GetString("database.dsn")).Msg("initializing database pool connection for the first time")
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

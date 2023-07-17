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

package common

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spf13/viper"
)

func SetupLogging() {
	// Set level
	level := viper.GetString("log.level")
	level = strings.ToLower(level)

	switch level {
	case "debug":
		log.Info().Msg("setting logging level to debug")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "error":
		log.Info().Msg("setting logging level to error")
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		log.Info().Msg("setting logging level to fatal")
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "info":
		log.Info().Msg("setting logging level to info")
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "panic":
		log.Info().Msg("setting logging level to panic")
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "trace":
		log.Info().Msg("setting logging level to trace")
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "warning":
		log.Info().Msg("setting logging level to warning")
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	// Set report caller
	if viper.GetBool("log.report_caller") {
		log.Logger = log.With().Caller().Logger()
	}

	// Setup output
	output := viper.GetString("log.output")
	switch output {
	case "stdout":
		if viper.GetBool("log.pretty") {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		} else {
			log.Logger = log.Output(os.Stdout)
		}
	case "stderr":
		if viper.GetBool("log.pretty") {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		} else {
			log.Logger = log.Output(os.Stderr)
		}
	default:
		fh, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		log.Logger = log.Output(fh)
	}

	// setup stack marshaler
	//nolint:reassign
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

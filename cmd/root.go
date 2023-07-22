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

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/go-geospatial/go-stac-server/common"
	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/middleware"
	"github.com/go-geospatial/go-stac-server/router"
	"github.com/go-geospatial/go-stac-server/static"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	json "github.com/goccy/go-json"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-stac-server",
	Short: "Serve STAC API v1.0.0",
	Long:  `go-stac-server implements a 1.0.0 compliant STAC api backed by a pgstac database`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		common.SetupLogging()
		log.Info().Msg("initialized logging")

		// try connecting to the database early so we fail fast if
		// we cannot connect to the database
		pool := database.GetInstance(ctx)
		defer pool.Close()
		log.Info().Msg("successfully connected to database")

		// Create new Fiber instance
		app := fiber.New(fiber.Config{
			JSONEncoder: json.Marshal,
			JSONDecoder: json.Unmarshal,
		})

		// shutdown cleanly on interrupt
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			sig := <-c // block until signal is read
			fmt.Printf("Received signal: '%s'; shutting down...\n", sig.String())
			err := app.ShutdownWithTimeout(time.Second * 5)
			if err != nil {
				log.Fatal().Err(err).Msg("app shutdown failed")
			}
		}()

		// Configure CORS
		corsConfig := cors.Config{
			AllowOrigins: "*",
			AllowHeaders: "Accept, Accept-CH, Accept-Charset, Accept-Datetime, Accept-Encoding, Accept-Ext, Accept-Features, Accept-Language, Accept-Params, Accept-Ranges, Access-Control-Allow-Credentials, Access-Control-Allow-Headers, Access-Control-Allow-Methods, Access-Control-Allow-Origin, Access-Control-Expose-Headers, Access-Control-Max-Age, Access-Control-Request-Headers, Access-Control-Request-Method, Age, Allow, Alternates, Authentication-Info, Authorization, C-Ext, C-Man, C-Opt, C-PEP, C-PEP-Info, CONNECT, Cache-Control, Compliance, Connection, Content-Base, Content-Disposition, Content-Encoding, Content-ID, Content-Language, Content-Length, Content-Location, Content-MD5, Content-Range, Content-Script-Type, Content-Security-Policy, Content-Style-Type, Content-Transfer-Encoding, Content-Type, Content-Version, Cookie, Cost, DAV, DELETE, DNT, DPR, Date, Default-Style, Delta-Base, Depth, Derived-From, Destination, Differential-ID, Digest, ETag, Expect, Expires, Ext, From, GET, GetProfile, HEAD, HTTP-date, Host, IM, If, If-Match, If-Modified-Since, If-None-Match, If-Range, If-Unmodified-Since, Keep-Alive, Label, Last-Event-ID, Last-Modified, Link, Location, Lock-Token, MIME-Version, Man, Max-Forwards, Media-Range, Message-ID, Meter, Negotiate, Non-Compliance, OPTION, OPTIONS, OWS, Opt, Optional, Ordering-Type, Origin, Overwrite, P3P, PEP, PICS-Label, POST, PUT, Pep-Info, Permanent, Position, Pragma, ProfileObject, Protocol, Protocol-Query, Protocol-Request, Proxy-Authenticate, Proxy-Authentication-Info, Proxy-Authorization, Proxy-Features, Proxy-Instruction, Public, RWS, Range, Referer, Refresh, Resolution-Hint, Resolver-Location, Retry-After, Safe, Sec-Websocket-Extensions, Sec-Websocket-Key, Sec-Websocket-Origin, Sec-Websocket-Protocol, Sec-Websocket-Version, Security-Scheme, Server, Set-Cookie, Set-Cookie2, SetProfile, SoapAction, Status, Status-URI, Strict-Transport-Security, SubOK, Subst, Surrogate-Capability, Surrogate-Control, TCN, TE, TRACE, Timeout, Title, Trailer, Transfer-Encoding, UA-Color, UA-Media, UA-Pixels, UA-Resolution, UA-Windowpixels, URI, Upgrade, User-Agent, Variant-Vary, Vary, Version, Via, Viewport-Width, WWW-Authenticate, Want-Digest, Warning, Width, X-Content-Duration, X-Content-Security-Policy, X-Content-Type-Options, X-CustomHeader, X-DNSPrefetch-Control, X-Forwarded-For, X-Forwarded-Port, X-Forwarded-Proto, X-Frame-Options, X-Modified, X-OTHER, X-PING, X-PINGOTHER, X-Powered-By, X-Requested-With",
			AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH",
		}
		app.Use(cors.New(corsConfig))

		// configure caching
		app.Use(cache.New(cache.Config{
			Next: func(c *fiber.Ctx) bool {
				return c.Query("refresh") == "true"
			},
			Expiration:   30 * time.Minute,
			CacheControl: true,
		}))

		// compression
		app.Use(compress.New(compress.Config{
			Level: compress.LevelBestSpeed, // 1
		}))

		// Setup logging middleware
		app.Use(middleware.NewLogger())

		// Add timing headers
		app.Use(middleware.Timer())

		prometheus := fiberprometheus.New("go-stac-server")
		prometheus.RegisterAt(app, "/metrics")
		app.Use(prometheus.Middleware)

		// Setup routes
		router.SetupRoutes(app)

		// configure static serves
		static.InitStaticFiles(app)

		err := app.Listen(":" + viper.GetString("server.port"))
		if err != nil {
			log.Fatal().Err(err).Msg("app.Listen returned an error")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-stac-server.yaml)")

	// server flags

	if err := viper.BindEnv("server.port", "PORT"); err != nil {
		log.Panic().Err(err).Msg("could not bind PORT")
	}
	rootCmd.Flags().IntP("port", "p", 3000, "Port to run application server on")
	if err := viper.BindPFlag("server.port", rootCmd.Flags().Lookup("port")); err != nil {
		log.Panic().Err(err).Msg("could not bind port")
	}

	// Logging configuration
	if err := viper.BindEnv("log.level", "LOG_LEVEL"); err != nil {
		log.Panic().Err(err).Msg("could not bind LOG_LEVEL")
	}
	rootCmd.PersistentFlags().String("log-level", "info", "Logging level")
	if err := viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		log.Panic().Err(err).Msg("could not bind log-level")
	}

	if err := viper.BindEnv("log.report_caller", "LOG_REPORT_CALLER"); err != nil {
		log.Panic().Err(err).Msg("could not bind LOG_REPORT_CALLER")
	}
	rootCmd.PersistentFlags().Bool("log-report-caller", false, "Log function name that called log statement")
	if err := viper.BindPFlag("log.report_caller", rootCmd.PersistentFlags().Lookup("log-report-caller")); err != nil {
		log.Panic().Err(err).Msg("could not bind log-report-caller")
	}

	if err := viper.BindEnv("log.output", "LOG_OUTPUT"); err != nil {
		log.Panic().Err(err).Msg("could not bind LOG_OUTPUT")
	}
	rootCmd.PersistentFlags().String("log-output", "stdout", "Write logs to specified output one of: file path, `stdout`, or `stderr`")
	if err := viper.BindPFlag("log.output", rootCmd.PersistentFlags().Lookup("log-output")); err != nil {
		log.Panic().Err(err).Msg("could not bind log-output")
	}

	if err := viper.BindEnv("log.otlp_url", "OTLP_URL"); err != nil {
		log.Panic().Err(err).Msg("could not bind OTLP_URL")
	}
	rootCmd.PersistentFlags().String("log-otlp-url", "", "OTLP server to send traces to, if blank don't send traces")
	if err := viper.BindPFlag("log.otlp_url", rootCmd.PersistentFlags().Lookup("log-otlp-url")); err != nil {
		log.Panic().Err(err).Msg("could not bind log-otlp-url")
	}

	// database
	if err := viper.BindEnv("database.dsn", "DSN"); err != nil {
		log.Panic().Err(err).Msg("could not bind DSN")
	}
	rootCmd.PersistentFlags().String("dsn", "", "PostgreSQL connection string")
	if err := viper.BindPFlag("database.dsn", rootCmd.PersistentFlags().Lookup("dsn")); err != nil {
		log.Panic().Err(err).Msg("could not bind database.dsn")
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "go-stac-server.toml"
		viper.AddConfigPath("/etc/")
		viper.AddConfigPath(fmt.Sprintf("%s/.config", home))
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("toml")
		viper.SetConfigName("go-stac-server.toml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info().Str("ConfigFile", viper.ConfigFileUsed()).Msg("Loaded config file")
	} else {
		log.Error().Stack().Err(err).Msg("error reading config file")
		os.Exit(1)
	}
}

package main

import (
	"oil/config"
	"oil/di"
	"oil/shared/logger"

	"github.com/rs/zerolog/log"

	migration "oil/helper"
)

// @title Oil API
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Get()

	logger.InitLogger()

	logger.SetLogLevel(cfg)

	if cfg.DB.Postgres.AutoMigrate {
		// Run migrations
		err := migration.Up(cfg)
		if err != nil {
			log.Error().Err(err).Msg("failed to run migrations")
		}
	}

	http, err := di.InitializeService()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize service")

		return
	}

	http.Serve()
}

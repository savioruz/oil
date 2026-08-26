// Package migrations provides database migration functions using goose.
package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"net"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
	"github.com/savioruz/oil/config"

	_ "github.com/lib/pq" // nolint:revive
)

const migrationsDir = "migrations/postgres"

func getDBName(config *config.Config, baseName string) string {
	if config.DB.Postgres.Prefix != "" {
		return config.DB.Postgres.Prefix + baseName
	}

	return baseName
}

func getConnection(config *config.Config) (*sql.DB, error) {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		config.DB.Postgres.Write.Username,
		config.DB.Postgres.Write.Password,
		net.JoinHostPort(config.DB.Postgres.Write.Host, config.DB.Postgres.Write.Port),
		getDBName(config, config.DB.Postgres.Write.Name),
		config.DB.Postgres.Write.SSLMode,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %w", err)
	}

	return db, nil
}

// Runner executes the specified migration action (up, down, step-up, drop) using the provided configuration.
func Runner(config *config.Config, action string) error {
	db, err := getConnection(config)
	if err != nil {
		return fmt.Errorf("error creating migration runner: %w", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("error closing database connection")
		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("error setting postgres dialect: %w", err)
	}

	goose.SetTableName(config.DB.Postgres.MigrationTable)

	switch action {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
			return fmt.Errorf("error running migrations: %w", err)
		}

		log.Info().Msg("Database migrations completed successfully")
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			return fmt.Errorf("error rolling back migrations: %w", err)
		}

		log.Info().Msg("Database migrations rolled back successfully")
	case "step-up":
		if err := goose.UpByOne(db, migrationsDir); err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
			return fmt.Errorf("error running migrations: %w", err)
		}

		log.Info().Msg("Database migrations completed successfully")
	case "drop":
		if err := goose.DownTo(db, migrationsDir, 0); err != nil {
			return fmt.Errorf("error rolling back migrations: %w", err)
		}

		log.Info().Msg("Database migrations rolled back successfully")
	}

	return nil
}

// Up applies all available migrations to the database using the provided configuration.
func Up(config *config.Config) error {
	return Runner(config, "up")
}

// StepUp applies the next available migration to the database using the provided configuration.
func StepUp(config *config.Config) error {
	return Runner(config, "step-up")
}

// Down rolls back the last applied migration from the database using the provided configuration.
func Down(config *config.Config) error {
	return Runner(config, "down")
}

// Drop rolls back all applied migrations from the database using the provided configuration.
func Drop(config *config.Config) error {
	return Runner(config, "drop")
}

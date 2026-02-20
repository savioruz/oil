// Package mocks provides mock implementations for postgres testing.
//
//nolint:revive
package mocks

import (
	"database/sql"
	"oil/infras/postgres"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// SetupPostgresConnection creates a mock postgres connection for testing.
func SetupPostgresConnection(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *postgres.Connection) {
	databases, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	sqlxDB := sqlx.NewDb(databases, "sqlmock")
	conn := &postgres.Connection{
		Read:  sqlxDB,
		Write: sqlxDB,
	}

	return databases, mock, conn
}

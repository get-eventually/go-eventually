package postgres

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

// RunMigrations runs the latest migrations for the postgres integration.
//
// Make sure to run these in the entrypoint of your application, ideally
// before building a postgres interface implementation.
func RunMigrations(db *sql.DB) error {
	wrapErr := func(err error, msg string) error {
		return fmt.Errorf("postgres.RunMigrations: %s, %w", msg, err)
	}

	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return wrapErr(err, "failed to create new iofs driver for reading migrations")
	}

	driver, err := pgx.WithInstance(db, &pgx.Config{
		MigrationsTable:       "eventually_schema_migrations",
		DatabaseName:          "",
		SchemaName:            "",
		StatementTimeout:      0,
		MigrationsTableQuoted: false,
		MultiStatementEnabled: false,
		MultiStatementMaxSize: 0,
	})
	if err != nil {
		return wrapErr(err, "failed to create new migrate db instance")
	}

	m, err := migrate.NewWithInstance("iofs", d, "pgx", driver)
	if err != nil {
		return wrapErr(err, "failed to create new migrate source for running db migrations")
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return wrapErr(err, "failed to execute migrations")
	}

	return nil
}

package postgres

import (
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	// Necessary to load the postgres driver used by migrate.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

// RunMigrations runs the latest migrations for the postgres integration.
//
// Make sure to run these in the entrypoint of your application, ideally
// before building a postgres interface implementation.
func RunMigrations(dsn string) error {
	wrapErr := func(err error, msg string) error {
		return fmt.Errorf("postgres.RunMigrations: %s, %w", msg, err)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return wrapErr(err, "invalid dsn format")
	}

	// go-migrate allows to specify a different migration table
	// than the default 'schema_migrations'. In this case, we want to use
	// a dedicated table to avoid potential clashing with the same tool running
	// on the same PostgreSQL database instance that is being used as
	// an Event Store.
	q := u.Query()
	q.Add("x-migrations-table", "eventually_schema_migrations")
	u.RawQuery = q.Encode()

	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return wrapErr(err, "failed to create new iofs driver for reading migrations")
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, u.String())
	if err != nil {
		return wrapErr(err, "failed to create new migrate source for running db migrations")
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return wrapErr(err, "failed to execute migrations")
	}

	return nil
}

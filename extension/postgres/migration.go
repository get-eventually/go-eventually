package postgres

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"

	"github.com/get-eventually/go-eventually/extension/postgres/migrations"
)

// RunMigrations performs the migrations for the Postgres database.
func RunMigrations(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("postgres.RunMigrations: invalid dsn format: %w", err)
	}

	// go-migrate allows to specify a different migration table
	// than the default 'schema_migrations'. In this case, we want to use
	// a dedicated table to avoid potential clashing with the same tool running
	// on the same PostgreSQL database instance that is being used as
	// an Event Store.
	q := u.Query()
	q.Add("x-migrations-table", "eventually_schema_migrations")
	u.RawQuery = q.Encode()

	src := bindata.Resource(migrations.AssetNames(), migrations.Asset)

	driver, err := bindata.WithInstance(src)
	if err != nil {
		return fmt.Errorf("postgres.RunMigrations: failed to access migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", driver, u.String())
	if err != nil {
		return fmt.Errorf("postgres.RunMigrations: failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres.RunMigrations: failed to migrate database: %w", err)
	}

	return nil
}

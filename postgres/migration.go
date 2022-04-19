package postgres

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

func RunMigrations(dsn string) error {
	wrapErr := func(err error, msg string) error {
		return fmt.Errorf("postgres.RunMigrations: %s, %w", msg, err)
	}

	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return wrapErr(err, "failed to create new iofs driver for reading migrations")
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return wrapErr(err, "failed to create new migrate source for running db migrations")
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return wrapErr(err, "failed to execute migrations")
	}

	return nil
}

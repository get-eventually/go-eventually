package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer returns an handle on a Postgres container
// started through testcontainers.
type PostgresContainer struct {
	*postgres.PostgresContainer

	ConnectionDSN  string
	PostgresConfig *pgx.ConnConfig
}

// NewPostgresContainer creates and starts a new Postgres container
// using testcontainers, then returns a handle to said container
// to manage its lifecycle.
func NewPostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	withContext := func(msg string, err error) error {
		return fmt.Errorf("internal.NewPostgresContainer: %s, %w", msg, err)
	}

	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("main"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("notasecret"),
		testcontainers.WithWaitStrategy(
			//nolint:mnd // It's ok to use a magic number here.
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return nil, withContext("failed to run new container", err)
	}

	dsn, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, withContext("failed to get connection dsn", err)
	}

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, withContext("failed to parse pgx config from dsn", err)
	}

	return &PostgresContainer{
		PostgresContainer: container,
		ConnectionDSN:     dsn,
		PostgresConfig:    config,
	}, nil
}

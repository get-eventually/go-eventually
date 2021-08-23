package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/postgres"
	"github.com/get-eventually/go-eventually/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const defaultPostgresURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

func obtainEventStore(t *testing.T) *postgres.EventStore {
	if testing.Short() {
		t.SkipNow()
	}

	url, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		url = defaultPostgresURL
	}

	require.NoError(t, postgres.RunMigrations(url))

	store, err := postgres.OpenEventStore(url)
	require.NoError(t, err)

	return store
}

func openDB() (*sql.DB, error) {
	url, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		url = defaultPostgresURL
	}

	return sql.Open("postgres", url)
}

func TestStoreSuite(t *testing.T) {
	store := obtainEventStore(t)
	defer func() { assert.NoError(t, store.Close()) }()

	handleError := func(err error) {
		if !assert.NoError(t, err) {
			t.FailNow()
		}
	}

	db, err := openDB()
	handleError(err)

	ctx := context.Background()
	handleError(store.Register(ctx, internal.IntPayload(0)))

	suite.Run(t, eventstore.NewStoreSuite(func() eventstore.Store {
		handleError := func(err error) {
			if !assert.NoError(t, err) {
				t.FailNow()
			}
		}

		tx, err := db.Begin()
		handleError(err)

		// Reset checkpoints for subscriptions.
		_, err = tx.Exec("DELETE FROM subscriptions_checkpoints")
		handleError(err)

		// Reset committed events and streams.
		_, err = tx.Exec("DELETE FROM streams")
		handleError(err)

		// Reset the global sequence number to 1.
		_, err = tx.Exec("ALTER SEQUENCE events_global_sequence_number_seq RESTART WITH 1")
		handleError(err)

		handleError(tx.Commit())

		return store
	}))
}

func TestRegister(t *testing.T) {
	store := obtainEventStore(t)
	defer func() { assert.NoError(t, store.Close()) }()

	t.Run("registering no events fails", func(t *testing.T) {
		err := store.Register(context.Background())
		assert.True(t, errors.Is(err, postgres.ErrEmptyEventsMap), "err", err)
	})

	t.Run("registering a nil event type fails", func(t *testing.T) {
		err := store.Register(context.Background(), nil)
		assert.Error(t, err)
	})

	t.Run("registering a type with event map shold be successful", func(t *testing.T) {
		assert.NoError(t,
			store.Register(context.Background(), internal.StringPayload("")),
		)
	})
}

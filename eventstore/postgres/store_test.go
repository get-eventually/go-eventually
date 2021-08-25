package postgres_test

import (
	"database/sql"
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

func obtainEventStore(t *testing.T) (*sql.DB, postgres.EventStore) {
	if testing.Short() {
		t.SkipNow()
	}

	url, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		url = defaultPostgresURL
	}

	require.NoError(t, postgres.RunMigrations(url))

	db, err := sql.Open("postgres", url)
	require.NoError(t, err)

	return db, postgres.NewEventStore(db)
}

func TestStoreSuite(t *testing.T) {
	db, store := obtainEventStore(t)
	defer func() { assert.NoError(t, db.Close()) }()

	require.NoError(t, store.Register(internal.IntPayload(0)))

	suite.Run(t, eventstore.NewStoreSuite(func() eventstore.Store {
		handleError := func(err error) {
			if !assert.NoError(t, err) {
				t.FailNow()
			}
		}

		tx, err := db.Begin()
		require.NoError(t, err)

		// Reset checkpoints for subscriptions.
		_, err = tx.Exec("DELETE FROM subscriptions_checkpoints")
		require.NoError(t, err)

		// Reset committed events and streams.
		_, err = tx.Exec("DELETE FROM streams")
		require.NoError(t, err)

		// Reset the global sequence number to 1.
		_, err = tx.Exec("ALTER SEQUENCE events_global_sequence_number_seq RESTART WITH 1")
		require.NoError(t, err)

		handleError(tx.Commit())

		return store
	}))
}

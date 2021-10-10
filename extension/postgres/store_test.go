package postgres_test

import (
	"context"
	"database/sql"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/postgres"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/internal"
)

var firstInstance = stream.ID{
	Type: "first-type",
	Name: "my-instance-for-latest-number",
}

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

func TestLatestSequenceNumber(t *testing.T) {
	db, store := obtainEventStore(t)
	defer func() { assert.NoError(t, db.Close()) }()

	require.NoError(t, store.Register(internal.IntPayload(0)))

	ctx := context.Background()

	for i := 1; i < 10; i++ {
		_, err := store.Append(
			ctx,
			firstInstance,
			eventstore.VersionCheck(int64(i-1)),
			eventually.Event{Payload: internal.IntPayload(i)},
		)

		require.NoError(t, err)
	}

	ch := make(chan eventstore.Event, 1)
	go func() {
		require.NoError(t, store.Stream(ctx, ch, stream.All{}, eventstore.SelectFromBeginning))
	}()

	var latestSequenceNumber int64
	for event := range ch {
		latestSequenceNumber = int64(math.Max(float64(latestSequenceNumber), float64(event.SequenceNumber)))
	}

	actual, err := store.LatestSequenceNumber(ctx)
	assert.NoError(t, err)
	assert.Equal(t, latestSequenceNumber, actual)
}

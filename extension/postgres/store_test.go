package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/event/version"
	"github.com/get-eventually/go-eventually/extension/postgres"
	"github.com/get-eventually/go-eventually/internal"
)

var firstInstance = event.StreamID{
	Type: "first-type",
	Name: "my-instance-for-latest-number",
}

const defaultPostgresURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

func obtainEventStore(t *testing.T) (*sql.DB, postgres.EventStore, postgres.Serde) {
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

	handleError := func(err error) {
		if !assert.NoError(t, err) {
			t.FailNow()
		}
	}

	tx, err := db.Begin()
	require.NoError(t, err)

	// Reset committed events and streams.
	_, err = tx.Exec("DELETE FROM streams")
	require.NoError(t, err)

	handleError(tx.Commit())

	serde := postgres.NewJSONRegistry()
	require.NoError(t, serde.Register(internal.IntPayload(0)))

	return db, postgres.NewEventStore(db, serde), serde
}

func TestStoreSuite(t *testing.T) {
	db, store, _ := obtainEventStore(t)
	defer func() { assert.NoError(t, db.Close()) }()

	suite.Run(t, event.NewStoreSuite(func() event.Store {
		return store
	}))
}

func TestAppendToStoreWrapperOption(t *testing.T) {
	db, _, serde := obtainEventStore(t)
	defer func() { assert.NoError(t, db.Close()) }()

	triggered := false

	store := postgres.NewEventStore(
		db,
		serde,
		postgres.WithAppendMiddleware(func(super postgres.AppendToStoreFunc) postgres.AppendToStoreFunc {
			return func(
				ctx context.Context,
				tx *sql.Tx,
				id event.StreamID,
				expected version.Check,
				eventName string,
				payload []byte,
				metadata []byte,
			) (version.Version, error) {
				triggered = true
				return super(ctx, tx, id, expected, eventName, payload, metadata)
			}
		}),
	)

	ctx := context.Background()

	newVersion, err := store.Append(
		ctx,
		firstInstance,
		version.Any,
		event.Event{Payload: internal.IntPayload(13)},
	)

	assert.NoError(t, err)
	assert.Equal(t, version.Version(1), newVersion)
	assert.True(t, triggered)
}

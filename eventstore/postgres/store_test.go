package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/eventually-rs/eventually-go/eventstore/postgres"

	"github.com/stretchr/testify/assert"
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

	store, err := postgres.OpenEventStore(url)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Cleanup(func() { assert.NoError(t, store.Close()) })

	return store
}

type testEvent struct {
	Value int64 `json:"value"`
}

func TestRegister(t *testing.T) {
	store := obtainEventStore(t)

	t.Run("registering a type with empty event map fails", func(t *testing.T) {
		err := store.Register(context.Background(), "register-fail-type", nil)
		assert.True(t, errors.Is(err, postgres.ErrEmptyEventsMap), "err", err)
	})

	t.Run("registering a type with event map shold be successful", func(t *testing.T) {
		assert.NoError(t,
			store.Register(context.Background(), "register-succes-type", map[string]interface{}{
				"test_event": testEvent{},
			}),
		)
	})
}

func TestCheckpointer(t *testing.T) {
	store := obtainEventStore(t)
	ctx := context.Background()

	const subscriptionName = "test-subscription"

	seqNum, err := store.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Zero(t, seqNum)

	newSeqNum := int64(1200)
	err = store.Write(ctx, subscriptionName, newSeqNum)
	assert.NoError(t, err)

	seqNum, err = store.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Equal(t, newSeqNum, seqNum)
}

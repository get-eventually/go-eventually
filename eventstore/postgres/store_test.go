package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/postgres"

	"github.com/stretchr/testify/assert"
)

const defaultPostgresURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

const (
	myType       = "my-type"
	myOtherType  = "my-other-type"
	testInstance = "test-instance"
)

var (
	expectedStreamAll = []eventstore.Event{
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: 1}.WithGlobalSequenceNumber(1),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: 1}.WithGlobalSequenceNumber(2),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: 2}.WithGlobalSequenceNumber(3),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: 2}.WithGlobalSequenceNumber(4),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: 3}.WithGlobalSequenceNumber(5),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: 3}.WithGlobalSequenceNumber(6),
		},
	}

	expectedStreamMyType = []eventstore.Event{
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: 1}.WithGlobalSequenceNumber(1),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: 2}.WithGlobalSequenceNumber(3),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: 3}.WithGlobalSequenceNumber(5),
		},
	}

	expectedStreamMyOtherType = []eventstore.Event{
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: 1}.WithGlobalSequenceNumber(2),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: 2}.WithGlobalSequenceNumber(4),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: 3}.WithGlobalSequenceNumber(6),
		},
	}
)

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

func TestEventStore_Stream(t *testing.T) {
	store := obtainEventStore(t)
	ctx := context.Background()

	// Register the stream types first.
	err := store.Register(ctx, myType, map[string]interface{}{
		"test_payload": int(0),
	})

	if !assert.NoError(t, err) {
		return
	}

	err = store.Register(ctx, myOtherType, map[string]interface{}{
		"test_other_payload": int(0),
	})

	if !assert.NoError(t, err) {
		return
	}

	// Get a typed access from the Event Store.
	myTypeStore, err := store.Type(ctx, myType)
	if !assert.NoError(t, err) {
		return
	}

	myOtherTypeStore, err := store.Type(ctx, myOtherType)
	if !assert.NoError(t, err) {
		return
	}

	// Make sure the Event Store is empty
	streamAll, err := eventstore.StreamToSlice(ctx, streamFromBeginning(store))
	assert.Empty(t, streamAll)
	assert.NoError(t, err)

	streamMyType, err := eventstore.StreamToSlice(ctx, streamFromBeginning(myTypeStore))
	assert.Empty(t, streamMyType)
	assert.NoError(t, err)

	streamMyOtherType, err := eventstore.StreamToSlice(ctx, streamFromBeginning(myOtherTypeStore))
	assert.Empty(t, streamMyOtherType)
	assert.NoError(t, err)

	// Append some events in both Streams
	for i := 1; i < 4; i++ {
		_, err = myTypeStore.
			Instance(testInstance).
			Append(ctx, int64(i-1), eventually.Event{Payload: i})

		assert.NoError(t, err)

		_, err = myOtherTypeStore.
			Instance(testInstance).
			Append(ctx, int64(i-1), eventually.Event{Payload: i})

		assert.NoError(t, err)
	}

	// Make sure the Event Store has recorded the events
	streamAll, err = eventstore.StreamToSlice(ctx, streamFromBeginning(store))
	assert.NoError(t, err)
	assert.Equal(t, expectedStreamAll, keepGlobalSequenceNumberOnly(streamAll))

	streamMyType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myTypeStore))
	assert.NoError(t, err)
	assert.Equal(t, expectedStreamMyType, keepGlobalSequenceNumberOnly(streamMyType))

	streamMyOtherType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myOtherTypeStore))
	assert.NoError(t, err)
	assert.Equal(t, expectedStreamMyOtherType, keepGlobalSequenceNumberOnly(streamMyOtherType))

	// Make sure the Instanced Event Store access also has recorded the events
	myTypeInstanceStore := myTypeStore.Instance(testInstance)
	myOtherTypeInstanceStore := myOtherTypeStore.Instance(testInstance)

	streamMyType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myTypeInstanceStore))
	assert.NoError(t, err)
	assert.Equal(t, expectedStreamMyType, keepGlobalSequenceNumberOnly(streamMyType))

	streamMyOtherType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myOtherTypeInstanceStore))
	assert.NoError(t, err)
	assert.Equal(t, expectedStreamMyOtherType, keepGlobalSequenceNumberOnly(streamMyOtherType))
}

func streamFromBeginning(streamer eventstore.Streamer) func(context.Context, eventstore.EventStream) error {
	return func(ctx context.Context, es eventstore.EventStream) error {
		return streamer.Stream(ctx, es, 0)
	}
}

func keepGlobalSequenceNumberOnly(events []eventstore.Event) []eventstore.Event {
	mapped := make([]eventstore.Event, 0, len(events))

	for _, event := range events {
		seqNum, _ := event.GlobalSequenceNumber()
		newEvent := event
		newEvent.Metadata = nil
		newEvent.Event = newEvent.WithGlobalSequenceNumber(seqNum)
		mapped = append(mapped, newEvent)
	}

	return mapped
}

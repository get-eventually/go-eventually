package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/postgres"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
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
			Event:      eventually.Event{Payload: intPayload(1)}.WithGlobalSequenceNumber(1),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: intPayload(1)}.WithGlobalSequenceNumber(2),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: intPayload(2)}.WithGlobalSequenceNumber(3),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: intPayload(2)}.WithGlobalSequenceNumber(4),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: intPayload(3)}.WithGlobalSequenceNumber(5),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: intPayload(3)}.WithGlobalSequenceNumber(6),
		},
	}

	expectedStreamMyType = []eventstore.Event{
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: intPayload(1)}.WithGlobalSequenceNumber(1),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: intPayload(2)}.WithGlobalSequenceNumber(3),
		},
		{
			StreamType: myType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: intPayload(3)}.WithGlobalSequenceNumber(5),
		},
	}

	expectedStreamMyOtherType = []eventstore.Event{
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    1,
			Event:      eventually.Event{Payload: intPayload(1)}.WithGlobalSequenceNumber(2),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    2,
			Event:      eventually.Event{Payload: intPayload(2)}.WithGlobalSequenceNumber(4),
		},
		{
			StreamType: myOtherType,
			StreamName: testInstance,
			Version:    3,
			Event:      eventually.Event{Payload: intPayload(3)}.WithGlobalSequenceNumber(6),
		},
	}
)

type intPayload int64

func (intPayload) Name() string { return "int_payload" }

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

	// Close connection on test exit
	t.Cleanup(func() { assert.NoError(t, store.Close()) })

	// Clean-up database on exit
	t.Cleanup(func() {
		handleError := func(err error) {
			if !assert.NoError(t, err) {
				t.FailNow()
			}
		}

		db, err := sql.Open("postgres", url)
		handleError(err)

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
	})

	return store
}

type testEvent struct {
	Value int64 `json:"value"`
}

func (testEvent) Name() string { return "test_event" }

func TestRegister(t *testing.T) {
	store := obtainEventStore(t)

	t.Run("registering a type with empty event map fails", func(t *testing.T) {
		err := store.Register(context.Background(), "register-fail-type", nil)
		assert.True(t, errors.Is(err, postgres.ErrEmptyEventsMap), "err", err)
	})

	t.Run("registering a type with event map shold be successful", func(t *testing.T) {
		assert.NoError(t,
			store.Register(context.Background(), "register-succes-type", testEvent{}),
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
	err := store.Register(ctx, myType, intPayload(0))
	if !assert.NoError(t, err) {
		return
	}

	err = store.Register(ctx, myOtherType, intPayload(0))
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
			Append(ctx, int64(i-1), eventually.Event{Payload: intPayload(i)})

		assert.NoError(t, err)

		_, err = myOtherTypeStore.
			Instance(testInstance).
			Append(ctx, int64(i-1), eventually.Event{Payload: intPayload(i)})

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

func TestEventStore_Subscribe(t *testing.T) {
	store := obtainEventStore(t)

	// To stop a Subscription, we need to explicitly cancel the context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register the stream types first.
	err := store.Register(ctx, myType, intPayload(0))
	if !assert.NoError(t, err) {
		return
	}

	err = store.Register(ctx, myOtherType, intPayload(0))
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

	// Open subscriptions for all interested Event Streams.
	// Channels are buffered with the same length as the length of the Event Stream,
	// optional only to avoid deadlocks in the test setting.
	streamAll := make(chan eventstore.Event, 6)
	streamMyType := make(chan eventstore.Event, 3)
	streamMyOtherType := make(chan eventstore.Event, 3)

	// Using a WaitGroup to synchronize writes only after all Subscriptions
	// have been opened.
	wg := new(sync.WaitGroup)
	wg.Add(3)

	subscribeTo := func(ctx context.Context, sub eventstore.Subscriber, es eventstore.EventStream) func() error {
		return func() error {
			wg.Done()
			return sub.Subscribe(ctx, es)
		}
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(subscribeTo(ctx, store, streamAll))
	group.Go(subscribeTo(ctx, myTypeStore, streamMyType))
	group.Go(subscribeTo(ctx, myOtherTypeStore, streamMyOtherType))

	// Append events in the Event Strems and then close the Subscriptions
	// by canceling the context.
	go func() {
		defer cancel()
		wg.Wait()

		for i := 1; i < 4; i++ {
			_, err := myTypeStore.
				Instance(testInstance).
				Append(ctx, int64(i-1), eventually.Event{Payload: intPayload(i)})

			assert.NoError(t, err)

			_, err = myOtherTypeStore.
				Instance(testInstance).
				Append(ctx, int64(i-1), eventually.Event{Payload: intPayload(i)})

			assert.NoError(t, err)
		}

		// NOTE: this is bad, I know, but we need to give time to the database
		// to send all the notifications to the subscriber.
		<-time.After(1 * time.Second)
	}()

	// Sink events from the Event Streams into slices for expectations comparison.
	streamedAll := keepGlobalSequenceNumberOnly(sinkToSlice(streamAll))
	streamedMyType := keepGlobalSequenceNumberOnly(sinkToSlice(streamMyType))
	streamedMyOtherType := keepGlobalSequenceNumberOnly(sinkToSlice(streamMyOtherType))

	assert.True(t, errors.Is(group.Wait(), context.Canceled))
	assert.Equal(t, expectedStreamAll, streamedAll)
	assert.Equal(t, expectedStreamMyType, streamedMyType)
	assert.Equal(t, expectedStreamMyOtherType, streamedMyOtherType)
}

func streamFromBeginning(streamer eventstore.Streamer) func(context.Context, eventstore.EventStream) error {
	return func(ctx context.Context, es eventstore.EventStream) error {
		return streamer.Stream(ctx, es, 0)
	}
}

func sinkToSlice(es <-chan eventstore.Event) []eventstore.Event {
	var events []eventstore.Event

	for event := range es {
		events = append(events, event)
	}

	return events
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

package inmemory_test

// import (
// 	"context"
// 	"errors"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/eventually-rs/eventually-go"
// 	"github.com/eventually-rs/eventually-go/eventstore"
// 	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
// 	"github.com/eventually-rs/eventually-go/internal"

// 	"github.com/stretchr/testify/assert"
// 	"golang.org/x/sync/errgroup"
// )

// const (
// 	myType       = "my-type"
// 	myOtherType  = "my-other-type"
// 	testInstance = "test-instance"
// )

// var (
// 	expectedStreamAll = []eventstore.Event{
// 		{
// 			StreamType: myType,
// 			StreamName: testInstance,
// 			Version:    1,
// 			Event:      eventually.Event{Payload: internal.IntPayload(1)}.WithGlobalSequenceNumber(1),
// 		},
// 		{
// 			StreamType: myOtherType,
// 			StreamName: testInstance,
// 			Version:    1,
// 			Event:      eventually.Event{Payload: internal.IntPayload(1)}.WithGlobalSequenceNumber(2),
// 		},
// 		{
// 			StreamType: myType,
// 			StreamName: testInstance,
// 			Version:    2,
// 			Event:      eventually.Event{Payload: internal.IntPayload(2)}.WithGlobalSequenceNumber(3),
// 		},
// 		{
// 			StreamType: myOtherType,
// 			StreamName: testInstance,
// 			Version:    2,
// 			Event:      eventually.Event{Payload: internal.IntPayload(2)}.WithGlobalSequenceNumber(4),
// 		},
// 		{
// 			StreamType: myType,
// 			StreamName: testInstance,
// 			Version:    3,
// 			Event:      eventually.Event{Payload: internal.IntPayload(3)}.WithGlobalSequenceNumber(5),
// 		},
// 		{
// 			StreamType: myOtherType,
// 			StreamName: testInstance,
// 			Version:    3,
// 			Event:      eventually.Event{Payload: internal.IntPayload(3)}.WithGlobalSequenceNumber(6),
// 		},
// 	}

// 	expectedStreamMyType = []eventstore.Event{
// 		{
// 			StreamType: myType,
// 			StreamName: testInstance,
// 			Version:    1,
// 			Event:      eventually.Event{Payload: internal.IntPayload(1)}.WithGlobalSequenceNumber(1),
// 		},
// 		{
// 			StreamType: myType,
// 			StreamName: testInstance,
// 			Version:    2,
// 			Event:      eventually.Event{Payload: internal.IntPayload(2)}.WithGlobalSequenceNumber(3),
// 		},
// 		{
// 			StreamType: myType,
// 			StreamName: testInstance,
// 			Version:    3,
// 			Event:      eventually.Event{Payload: internal.IntPayload(3)}.WithGlobalSequenceNumber(5),
// 		},
// 	}

// 	expectedStreamMyOtherType = []eventstore.Event{
// 		{
// 			StreamType: myOtherType,
// 			StreamName: testInstance,
// 			Version:    1,
// 			Event:      eventually.Event{Payload: internal.IntPayload(1)}.WithGlobalSequenceNumber(2),
// 		},
// 		{
// 			StreamType: myOtherType,
// 			StreamName: testInstance,
// 			Version:    2,
// 			Event:      eventually.Event{Payload: internal.IntPayload(2)}.WithGlobalSequenceNumber(4),
// 		},
// 		{
// 			StreamType: myOtherType,
// 			StreamName: testInstance,
// 			Version:    3,
// 			Event:      eventually.Event{Payload: internal.IntPayload(3)}.WithGlobalSequenceNumber(6),
// 		},
// 	}
// )

// func TestType(t *testing.T) {
// 	t.Run("fail when getting a Typed instance but type has not been registered", func(t *testing.T) {
// 		store := inmemory.NewEventStore()
// 		typ, err := store.Type(context.Background(), "test-type")
// 		assert.Nil(t, typ)
// 		assert.Error(t, err)
// 	})

// 	t.Run("Typed instance is returned if type has been registered", func(t *testing.T) {
// 		store := inmemory.NewEventStore()
// 		ctx := context.Background()
// 		typeName := "test-type"

// 		// Note: the inmemory event store does not require events to be
// 		// explicitly registered, as there is no deserialization involved.
// 		if err := store.Register(ctx, typeName, nil); !assert.NoError(t, err) {
// 			return
// 		}

// 		typ, err := store.Type(context.Background(), "test-type")
// 		assert.NotNil(t, typ)
// 		assert.NoError(t, err)
// 	})
// }

// func TestEventStore_Stream(t *testing.T) {
// 	ctx := context.Background()
// 	store := inmemory.NewEventStore()

// 	// Register the stream types first.
// 	if err := store.Register(ctx, myType, nil); !assert.NoError(t, err) {
// 		return
// 	}

// 	if err := store.Register(ctx, myOtherType, nil); !assert.NoError(t, err) {
// 		return
// 	}

// 	// Get a typed access from the Event Store.
// 	myTypeStore, err := store.Type(ctx, myType)
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	myOtherTypeStore, err := store.Type(ctx, myOtherType)
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	// Make sure the Event Store is empty
// 	streamAll, err := eventstore.StreamToSlice(ctx, streamFromBeginning(store))
// 	assert.Empty(t, streamAll)
// 	assert.NoError(t, err)

// 	streamMyType, err := eventstore.StreamToSlice(ctx, streamFromBeginning(myTypeStore))
// 	assert.Empty(t, streamMyType)
// 	assert.NoError(t, err)

// 	streamMyOtherType, err := eventstore.StreamToSlice(ctx, streamFromBeginning(myOtherTypeStore))
// 	assert.Empty(t, streamMyOtherType)
// 	assert.NoError(t, err)

// 	// Append some events in both Streams
// 	for i := 1; i < 4; i++ {
// 		_, err = myTypeStore.
// 			Instance(testInstance).
// 			Append(ctx, int64(i-1), eventually.Event{Payload: internal.IntPayload(i)})

// 		assert.NoError(t, err)

// 		_, err = myOtherTypeStore.
// 			Instance(testInstance).
// 			Append(ctx, int64(i-1), eventually.Event{Payload: internal.IntPayload(i)})

// 		assert.NoError(t, err)
// 	}

// 	// Make sure the Event Store has recorded the events
// 	streamAll, err = eventstore.StreamToSlice(ctx, streamFromBeginning(store))
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedStreamAll, streamAll)

// 	streamMyType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myTypeStore))
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedStreamMyType, streamMyType)

// 	streamMyOtherType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myOtherTypeStore))
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedStreamMyOtherType, streamMyOtherType)

// 	// Make sure the Instanced Event Store access also has recorded the events
// 	myTypeInstanceStore := myTypeStore.Instance(testInstance)
// 	myOtherTypeInstanceStore := myOtherTypeStore.Instance(testInstance)

// 	streamMyType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myTypeInstanceStore))
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedStreamMyType, streamMyType)

// 	streamMyOtherType, err = eventstore.StreamToSlice(ctx, streamFromBeginning(myOtherTypeInstanceStore))
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedStreamMyOtherType, streamMyOtherType)
// }

// func TestEventStore_Subscribe(t *testing.T) {
// 	// To stop a Subscription, we need to explicitly cancel the context.
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	store := inmemory.NewEventStore()

// 	// Register the stream types first.
// 	if err := store.Register(ctx, myType, nil); !assert.NoError(t, err) {
// 		return
// 	}

// 	if err := store.Register(ctx, myOtherType, nil); !assert.NoError(t, err) {
// 		return
// 	}

// 	// Get a typed access from the Event Store.
// 	myTypeStore, err := store.Type(ctx, myType)
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	myOtherTypeStore, err := store.Type(ctx, myOtherType)
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	// Open subscriptions for all interested Event Streams.
// 	// Channels are buffered with the same length as the length of the Event Stream,
// 	// optional only to avoid deadlocks in the test setting.
// 	streamAll := make(chan eventstore.Event, 6)
// 	streamMyType := make(chan eventstore.Event, 3)
// 	streamMyOtherType := make(chan eventstore.Event, 3)

// 	// Using a WaitGroup to synchronize writes only after all Subscriptions
// 	// have been opened.
// 	wg := new(sync.WaitGroup)
// 	wg.Add(3)

// 	subscribeTo := func(ctx context.Context, sub eventstore.Subscriber, es eventstore.EventStream) func() error {
// 		return func() error {
// 			go func() { wg.Done() }()
// 			return sub.Subscribe(ctx, es)
// 		}
// 	}

// 	group, ctx := errgroup.WithContext(ctx)
// 	group.Go(subscribeTo(ctx, store, streamAll))
// 	group.Go(subscribeTo(ctx, myTypeStore, streamMyType))
// 	group.Go(subscribeTo(ctx, myOtherTypeStore, streamMyOtherType))

// 	// Append events in the Event Strems and then close the Subscriptions
// 	// by canceling the context.
// 	go func() {
// 		defer cancel()
// 		wg.Wait()

// 		for i := 1; i < 4; i++ {
// 			_, err := myTypeStore.
// 				Instance(testInstance).
// 				Append(ctx, int64(i-1), eventually.Event{Payload: internal.IntPayload(i)})

// 			assert.NoError(t, err)

// 			_, err = myOtherTypeStore.
// 				Instance(testInstance).
// 				Append(ctx, int64(i-1), eventually.Event{Payload: internal.IntPayload(i)})

// 			assert.NoError(t, err)
// 		}

// 		<-time.After(100 * time.Millisecond)
// 	}()

// 	// Sink events from the Event Streams into slices for expectations comparison.
// 	streamedAll := sinkToSlice(streamAll)
// 	streamedMyType := sinkToSlice(streamMyType)
// 	streamedMyOtherType := sinkToSlice(streamMyOtherType)

// 	assert.True(t, errors.Is(group.Wait(), context.Canceled))
// 	assert.Equal(t, expectedStreamAll, streamedAll)
// 	assert.Equal(t, expectedStreamMyType, streamedMyType)
// 	assert.Equal(t, expectedStreamMyOtherType, streamedMyOtherType)
// }

// func streamFromBeginning(streamer eventstore.Streamer) func(context.Context, eventstore.EventStream) error {
// 	return func(ctx context.Context, es eventstore.EventStream) error {
// 		return streamer.Stream(ctx, es, 0)
// 	}
// }

// func sinkToSlice(es <-chan eventstore.Event) []eventstore.Event {
// 	var events []eventstore.Event

// 	for event := range es {
// 		events = append(events, event)
// 	}

// 	return events
// }

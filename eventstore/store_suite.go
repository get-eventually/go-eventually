package eventstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
)

var (
	firstInstance = StreamID{
		Type: "first-type",
		Name: "my-instance",
	}

	secondInstance = StreamID{
		Type: "second-type",
		Name: "my-instance",
	}

	expectedStreamAll = []Event{
		{
			Stream:         firstInstance,
			Version:        1,
			SequenceNumber: 1,
			Event:          eventually.Event{Payload: internal.IntPayload(1)},
		},
		{
			Stream:         secondInstance,
			Version:        1,
			SequenceNumber: 2,
			Event:          eventually.Event{Payload: internal.IntPayload(1)},
		},
		{
			Stream:         firstInstance,
			Version:        2,
			SequenceNumber: 3,
			Event:          eventually.Event{Payload: internal.IntPayload(2)},
		},
		{
			Stream:         secondInstance,
			Version:        2,
			SequenceNumber: 4,
			Event:          eventually.Event{Payload: internal.IntPayload(2)},
		},
		{
			Stream:         firstInstance,
			Version:        3,
			SequenceNumber: 5,
			Event:          eventually.Event{Payload: internal.IntPayload(3)},
		},
		{
			Stream:         secondInstance,
			Version:        3,
			SequenceNumber: 6,
			Event:          eventually.Event{Payload: internal.IntPayload(3)},
		},
	}

	expectedStreamFirstInstance = []Event{
		{
			Stream:         firstInstance,
			Version:        1,
			SequenceNumber: 1,
			Event:          eventually.Event{Payload: internal.IntPayload(1)},
		},
		{
			Stream:         firstInstance,
			Version:        2,
			SequenceNumber: 3,
			Event:          eventually.Event{Payload: internal.IntPayload(2)},
		},
		{
			Stream:         firstInstance,
			Version:        3,
			SequenceNumber: 5,
			Event:          eventually.Event{Payload: internal.IntPayload(3)},
		},
	}

	expectedStreamSecondInstance = []Event{
		{
			Stream:         secondInstance,
			Version:        1,
			SequenceNumber: 2,
			Event:          eventually.Event{Payload: internal.IntPayload(1)},
		},
		{
			Stream:         secondInstance,
			Version:        2,
			SequenceNumber: 4,
			Event:          eventually.Event{Payload: internal.IntPayload(2)},
		},
		{
			Stream:         secondInstance,
			Version:        3,
			SequenceNumber: 6,
			Event:          eventually.Event{Payload: internal.IntPayload(3)},
		},
	}
)

// StoreSuite is a full testing suite for an eventstore.Store instance.
type StoreSuite struct {
	suite.Suite

	storeFactory func() Store
	eventStore   Store // NOTE: this instance is initialized in SetupTest.
}

// NewStoreSuite creates a new Event Store testing suite using the provided
// eventstore.Store type.
func NewStoreSuite(factory func() Store) *StoreSuite {
	ss := new(StoreSuite)
	ss.storeFactory = factory

	return ss
}

// SetupTest creates a new, fresh Event Store instance for each test in the suite.
func (ss *StoreSuite) SetupTest() {
	ss.eventStore = ss.storeFactory()
}

// TestStream tests all the eventstore.Streamer functions using the provided
// Event Store instance.
func (ss *StoreSuite) TestStream() {
	t := ss.T()
	ctx := context.Background()

	// Make sure the Event Store is completely empty.
	streamAll, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamAll(ctx, es, SelectFromBeginning)
	})

	assert.Empty(t, streamAll)

	if !assert.NoError(t, err) {
		return
	}

	// Append some events for the two test event stream types.
	if err = ss.appendEvents(ctx); !assert.NoError(t, err) {
		return
	}

	// Make sure the Event Store has recorded the events as expected.
	streamAll, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamAll(ctx, es, SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamFirstType, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamByType(ctx, es, firstInstance.Type, SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamSecondType, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamByType(ctx, es, secondInstance.Type, SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamFirstInstance, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, firstInstance, SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamSecondInstance, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, secondInstance, SelectFromBeginning)
	})

	assert.NoError(t, err)

	assert.Equal(t, expectedStreamAll, ss.skipMetadata(streamAll))
	assert.Equal(t, expectedStreamFirstInstance, ss.skipMetadata(streamFirstType))
	assert.Equal(t, expectedStreamFirstInstance, ss.skipMetadata(streamFirstInstance))
	assert.Equal(t, expectedStreamSecondInstance, ss.skipMetadata(streamSecondType))
	assert.Equal(t, expectedStreamSecondInstance, ss.skipMetadata(streamSecondInstance))

	// Streaming with an out-of-bound Select will yield empty elements.
	streamAll, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamAll(ctx, es, Select{From: 7})
	})

	assert.NoError(t, err)

	streamFirstType, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamByType(ctx, es, firstInstance.Type, Select{From: 7})
	})

	assert.NoError(t, err)

	streamSecondType, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.StreamByType(ctx, es, secondInstance.Type, Select{From: 7})
	})

	assert.NoError(t, err)

	streamFirstInstance, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, firstInstance, Select{From: 4})
	})

	assert.NoError(t, err)

	streamSecondInstance, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, secondInstance, Select{From: 4})
	})

	assert.NoError(t, err)

	assert.Empty(t, streamAll)
	assert.Empty(t, streamFirstType)
	assert.Empty(t, streamSecondType)
	assert.Empty(t, streamFirstInstance)
	assert.Empty(t, streamSecondInstance)
}

// TestSubscribe tests all the eventstore.Subscriber functions using the provided
// Event Store instance.
func (ss *StoreSuite) TestSubscribe() {
	t := ss.T()

	// To stop a Subscriber, we need to explicitly cancel the context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Open subscribers for all interested Event Streams.
	// Channels are buffered with the same length as the length of the Event Stream,
	// optional only to avoid deadlocks in the test setting.
	streamAll := make(chan Event, len(expectedStreamAll))
	streamFirstInstance := make(chan Event, len(expectedStreamFirstInstance))
	streamSecondInstance := make(chan Event, len(expectedStreamSecondInstance))

	// Using a WaitGroup to synchronize writes only after all Subscriptions
	// have been opened.
	wg := new(sync.WaitGroup)
	wg.Add(3)

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		wg.Done()
		return ss.eventStore.SubscribeToAll(ctx, streamAll)
	})

	group.Go(func() error {
		wg.Done()
		return ss.eventStore.SubscribeToType(ctx, streamFirstInstance, firstInstance.Type)
	})

	group.Go(func() error {
		wg.Done()
		return ss.eventStore.SubscribeToType(ctx, streamSecondInstance, secondInstance.Type)
	})

	wg.Wait()

	group.Go(func() error {
		defer cancel()

		// NOTE: this is bad, I know, but we need to give time to the database
		// to send all the notifications to the subscriber.
		<-time.After(500 * time.Millisecond)

		if err := ss.appendEvents(ctx); err != nil {
			return err
		}

		// NOTE: this is also bad, but same reason as above :shrugs:
		<-time.After(500 * time.Millisecond)

		return nil
	})

	if err := group.Wait(); !assert.True(t, errors.Is(err, context.Canceled), "err", err) {
		return
	}

	// Sink events from the Event Streams into slices for expectations comparison.
	streamedAll := ss.skipMetadata(ss.sinkToSlice(streamAll))
	streamedFirstInstance := ss.skipMetadata(ss.sinkToSlice(streamFirstInstance))
	streamedSecondInstance := ss.skipMetadata(ss.sinkToSlice(streamSecondInstance))

	assert.Equal(t, expectedStreamAll, streamedAll)
	assert.Equal(t, expectedStreamFirstInstance, streamedFirstInstance)
	assert.Equal(t, expectedStreamSecondInstance, streamedSecondInstance)
}

func (ss *StoreSuite) appendEvents(ctx context.Context) error {
	for i := 1; i < 4; i++ {
		if _, err := ss.eventStore.Append(
			ctx,
			firstInstance,
			VersionCheck(int64(i-1)),
			eventually.Event{Payload: internal.IntPayload(i)},
		); err != nil {
			return fmt.Errorf("appendEvents: failed on first instance, event %d: %w", i, err)
		}

		if _, err := ss.eventStore.Append(
			ctx,
			secondInstance,
			VersionCheck(int64(i-1)),
			eventually.Event{Payload: internal.IntPayload(i)},
		); err != nil {
			return fmt.Errorf("appendEvents: failed on second instance, event %d: %w", i, err)
		}
	}

	return nil
}

func (*StoreSuite) sinkToSlice(es <-chan Event) []Event {
	var events []Event

	for event := range es {
		events = append(events, event)
	}

	return events
}

func (*StoreSuite) skipMetadata(events []Event) []Event {
	mapped := make([]Event, 0, len(events))

	for _, event := range events {
		newEvent := event
		newEvent.Metadata = nil
		mapped = append(mapped, newEvent)
	}

	return mapped
}

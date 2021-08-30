package eventstore

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/internal"
)

var (
	firstInstance = stream.ID{
		Type: "first-type",
		Name: "my-instance",
	}

	secondInstance = stream.ID{
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
		return ss.eventStore.Stream(ctx, es, stream.All{}, SelectFromBeginning)
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
		return ss.eventStore.Stream(ctx, es, stream.All{}, SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamAllByTypes, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByTypes{firstInstance.Type, secondInstance.Type}, SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamFirstType, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByType(firstInstance.Type), SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamSecondType, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByType(secondInstance.Type), SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamFirstInstance, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByID(firstInstance), SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamSecondInstance, err := StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByID(secondInstance), SelectFromBeginning)
	})

	assert.NoError(t, err)

	assert.Equal(t, expectedStreamAll, ss.skipMetadata(streamAll))
	assert.Equal(t, expectedStreamAll, ss.skipMetadata(streamAllByTypes))
	assert.Equal(t, expectedStreamFirstInstance, ss.skipMetadata(streamFirstType))
	assert.Equal(t, expectedStreamFirstInstance, ss.skipMetadata(streamFirstInstance))
	assert.Equal(t, expectedStreamSecondInstance, ss.skipMetadata(streamSecondType))
	assert.Equal(t, expectedStreamSecondInstance, ss.skipMetadata(streamSecondInstance))

	// Streaming with an out-of-bound Select will yield empty elements.
	streamAll, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.All{}, Select{From: 7})
	})

	assert.NoError(t, err)

	streamAllByTypes, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByTypes{firstInstance.Type, secondInstance.Type}, Select{From: 7})
	})

	assert.NoError(t, err)

	streamFirstType, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByType(firstInstance.Type), Select{From: 7})
	})

	assert.NoError(t, err)

	streamSecondType, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByType(secondInstance.Type), Select{From: 7})
	})

	assert.NoError(t, err)

	streamFirstInstance, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByID(firstInstance), Select{From: 4})
	})

	assert.NoError(t, err)

	streamSecondInstance, err = StreamToSlice(ctx, func(ctx context.Context, es EventStream) error {
		return ss.eventStore.Stream(ctx, es, stream.ByID(secondInstance), Select{From: 4})
	})

	assert.NoError(t, err)

	assert.Empty(t, streamAll)
	assert.Empty(t, streamAllByTypes)
	assert.Empty(t, streamFirstType)
	assert.Empty(t, streamSecondType)
	assert.Empty(t, streamFirstInstance)
	assert.Empty(t, streamSecondInstance)
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

func (*StoreSuite) skipMetadata(events []Event) []Event {
	mapped := make([]Event, 0, len(events))

	for _, event := range events {
		newEvent := event
		newEvent.Metadata = nil
		mapped = append(mapped, newEvent)
	}

	return mapped
}

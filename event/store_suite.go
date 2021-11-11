package event

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/get-eventually/go-eventually/event/version"
	"github.com/get-eventually/go-eventually/internal"
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

	thirdInstance = StreamID{
		Type: "third-type",
		Name: "my-instance",
	}

	expectedStreamFirstInstance = []Persisted{
		{
			Stream:         firstInstance,
			Version:        1,
			SequenceNumber: 1,
			Event:          Event{Payload: internal.IntPayload(1)},
		},
		{
			Stream:         firstInstance,
			Version:        2,
			SequenceNumber: 3,
			Event:          Event{Payload: internal.IntPayload(2)},
		},
		{
			Stream:         firstInstance,
			Version:        3,
			SequenceNumber: 5,
			Event:          Event{Payload: internal.IntPayload(3)},
		},
	}

	expectedStreamSecondInstance = []Persisted{
		{
			Stream:         secondInstance,
			Version:        1,
			SequenceNumber: 2,
			Event:          Event{Payload: internal.IntPayload(1)},
		},
		{
			Stream:         secondInstance,
			Version:        2,
			SequenceNumber: 4,
			Event:          Event{Payload: internal.IntPayload(2)},
		},
		{
			Stream:         secondInstance,
			Version:        3,
			SequenceNumber: 6,
			Event:          Event{Payload: internal.IntPayload(3)},
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
func (ss *StoreSuite) TestStore() {
	t := ss.T()
	ctx := context.Background()

	// Append some events for the two test event stream types.
	if err := ss.appendEvents(ctx); !assert.NoError(t, err) {
		return
	}

	// Make sure the Event Store has recorded the events as expected.
	streamFirstInstance, err := StreamToSlice(ctx, func(ctx context.Context, es Stream) error {
		return ss.eventStore.Stream(ctx, es, firstInstance, version.SelectFromBeginning)
	})

	assert.NoError(t, err)

	streamSecondInstance, err := StreamToSlice(ctx, func(ctx context.Context, es Stream) error {
		return ss.eventStore.Stream(ctx, es, secondInstance, version.SelectFromBeginning)
	})

	assert.NoError(t, err)

	assert.Equal(t, expectedStreamFirstInstance, ss.skipMetadata(streamFirstInstance))
	assert.Equal(t, expectedStreamSecondInstance, ss.skipMetadata(streamSecondInstance))

	// Streaming with an out-of-bound Select will yield empty elements.
	streamFirstInstance, err = StreamToSlice(ctx, func(ctx context.Context, es Stream) error {
		return ss.eventStore.Stream(ctx, es, firstInstance, version.Selector{From: 4})
	})

	assert.NoError(t, err)

	streamSecondInstance, err = StreamToSlice(ctx, func(ctx context.Context, es Stream) error {
		return ss.eventStore.Stream(ctx, es, secondInstance, version.Selector{From: 4})
	})

	assert.NoError(t, err)

	assert.Empty(t, streamFirstInstance)
	assert.Empty(t, streamSecondInstance)

	// Testing eventstore.Appender interface and the optimistic concurrency handling.

	newVersion, err := ss.eventStore.Append(
		ctx,
		thirdInstance,
		version.CheckExact(0), // No event expected on this Event Stream!
		Event{Payload: internal.IntPayload(0)},
	)

	assert.Equal(t, version.Version(1), newVersion)
	assert.NoError(t, err)

	_, err = ss.eventStore.Append(
		ctx,
		thirdInstance,
		version.CheckExact(0), // Appending with the same expected version should fail!
		Event{Payload: internal.IntPayload(0)},
	)

	expectedErr := version.ConflictError{
		Expected: 0,
		Actual:   1,
	}

	var actualErr version.ConflictError

	assert.ErrorAs(t, err, &actualErr)
	assert.Equal(t, expectedErr, actualErr)
}

func (ss *StoreSuite) appendEvents(ctx context.Context) error {
	for i := 1; i < 4; i++ {
		if _, err := ss.eventStore.Append(
			ctx,
			firstInstance,
			version.CheckExact(version.Version(i-1)),
			Event{Payload: internal.IntPayload(i)},
		); err != nil {
			return fmt.Errorf("appendEvents: failed on first instance, event %d: %w", i, err)
		}

		if _, err := ss.eventStore.Append(
			ctx,
			secondInstance,
			version.CheckExact(version.Version(i-1)),
			Event{Payload: internal.IntPayload(i)},
		); err != nil {
			return fmt.Errorf("appendEvents: failed on second instance, event %d: %w", i, err)
		}
	}

	return nil
}

func (*StoreSuite) skipMetadata(events []Persisted) []Persisted {
	mapped := make([]Persisted, 0, len(events))

	for _, event := range events {
		newEvent := event
		newEvent.Metadata = nil
		mapped = append(mapped, newEvent)
	}

	return mapped
}

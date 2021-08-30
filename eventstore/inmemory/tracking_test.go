package inmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/internal"
)

func TestTrackingEventStore(t *testing.T) {
	t.Run("no events recorded when no events are appended", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)
		assert.Empty(t, trackingEventStore.Recorded())
	})

	t.Run("events recorded successfully", func(t *testing.T) {
		ctx := context.Background()
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)

		streamID := stream.ID{
			Type: "test-type",
			Name: "test-instance",
		}

		_, err := trackingEventStore.Append(
			ctx,
			streamID,
			eventstore.VersionCheck(0),
			eventually.Event{Payload: internal.StringPayload("hello")},
			eventually.Event{Payload: internal.StringPayload("world")},
		)

		if !assert.NoError(t, err) {
			return
		}

		// Compare events in the event store and recorded ones from the tracking store.
		events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			return eventStore.Stream(ctx, es, stream.ByID(streamID), eventstore.SelectFromBeginning)
		})

		assert.NoError(t, err)
		assert.Equal(t, stripMetadata(events), trackingEventStore.Recorded())
	})
}

func stripMetadata(events []eventstore.Event) []eventstore.Event {
	for i := range events {
		events[i].SequenceNumber = 0
		events[i].Metadata = nil
	}

	return events
}

package inmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/event/version"
	"github.com/get-eventually/go-eventually/extension/inmemory"
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

		streamID := event.StreamID{
			Type: "test-type",
			Name: "test-instance",
		}

		_, err := trackingEventStore.Append(
			ctx,
			streamID,
			version.CheckExact(0),
			event.Event{Payload: internal.StringPayload("hello")},
			event.Event{Payload: internal.StringPayload("world")},
		)

		if !assert.NoError(t, err) {
			return
		}

		// Compare events in the event store and recorded ones from the tracking store.
		events, err := event.StreamToSlice(ctx, func(ctx context.Context, eventStream event.Stream) error {
			return eventStore.Stream(ctx, eventStream, streamID, version.SelectFromBeginning)
		})

		assert.NoError(t, err)
		assert.Equal(t, stripMetadata(events), trackingEventStore.Recorded())
	})
}

func stripMetadata(events []event.Persisted) []event.Persisted {
	for i := range events {
		events[i].Metadata = nil
	}

	return events
}

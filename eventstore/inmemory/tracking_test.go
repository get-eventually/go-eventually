package inmemory_test

import (
	"context"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"

	"github.com/stretchr/testify/assert"
)

type stringPayload string

func (stringPayload) Name() string { return "string_payload" }

func TestTrackingEventStore(t *testing.T) {
	t.Run("no events recorded when no events are appended", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := &inmemory.TrackingEventStore{Store: eventStore}
		assert.Empty(t, trackingEventStore.Recorded())
	})

	t.Run("events recorded successfully", func(t *testing.T) {
		const testType = "test-type"
		const testInstance = "test-instance"

		ctx := context.Background()
		eventStore := inmemory.NewEventStore()
		trackingEventStore := &inmemory.TrackingEventStore{Store: eventStore}

		// Register stream type for appending events.
		// Must use the tracking event store to start recording committed events.
		if err := trackingEventStore.Register(ctx, testType, nil); !assert.NoError(t, err) {
			return
		}

		trackingTypedStore, err := trackingEventStore.Type(ctx, testType)
		if !assert.NoError(t, err) {
			return
		}

		_, err = trackingTypedStore.
			Instance(testInstance).
			Append(ctx, 0,
				eventually.Event{Payload: stringPayload("hello")},
				eventually.Event{Payload: stringPayload("world")},
			)

		if !assert.NoError(t, err) {
			return
		}

		// Compare events in the event store and recorded ones from the tracking store.
		events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			return eventStore.Stream(ctx, es, 0)
		})

		assert.NoError(t, err)
		assert.ElementsMatch(t, stripMetadata(events), trackingEventStore.Recorded())
	})
}

func stripMetadata(events []eventstore.Event) []eventstore.Event {
	for i := range events {
		events[i].Metadata = nil
	}

	return events
}

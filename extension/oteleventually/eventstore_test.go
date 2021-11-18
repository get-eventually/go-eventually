package oteleventually_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/extension/oteleventually"
	"github.com/get-eventually/go-eventually/internal"
)

func TestInstrumentedEventStore_Append(t *testing.T) {
	baseEventStore := inmemory.NewEventStore()
	trackingAppender := inmemory.NewTrackingEventStore(baseEventStore)
	trackingEventStore := eventstore.Fused{
		Appender: trackingAppender,
		Streamer: baseEventStore,
	}

	eventStore, err := oteleventually.InstrumentEventStore(trackingEventStore)
	require.NoError(t, err)

	ctx, span := otel.GetTracerProvider().Tracer(t.Name()).Start(context.Background(), t.Name())
	defer span.End()

	streamID := stream.ID{
		Type: "test-type",
		Name: "test-name",
	}

	evt := eventually.Event{
		Payload:  internal.IntPayload(0),
		Metadata: map[string]interface{}{},
	}

	eventStreamVersion, err := eventStore.Append(ctx, streamID, eventstore.VersionCheck(0), evt)
	assert.Equal(t, int64(1), eventStreamVersion)
	assert.NoError(t, err)

	events := trackingAppender.Recorded()
	assert.Len(t, events, 1)

	recordedEvent := events[0]
	assert.Equal(t, span.SpanContext().TraceID().String(), recordedEvent.Metadata["X-Eventually-TraceId"])
}

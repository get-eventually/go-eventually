package eventstore_test

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

func TestContextAware(t *testing.T) {
	eventStore := inmemory.NewEventStore()
	trackingEventStore := inmemory.NewTrackingEventStore(eventStore)
	contextAwareEventStore := eventstore.NewContextAware(trackingEventStore)

	metadata := eventually.Metadata{
		"Test":  "value",
		"Test2": 1,
	}

	streamID := stream.ID{
		Type: "test-type",
		Name: "test-name",
	}

	event := eventually.Event{
		Payload:  internal.IntPayload(0),
		Metadata: eventually.Metadata{},
	}

	ctx := eventstore.ContextMetadata(context.Background(), metadata)
	newEventStreamVersion, err := contextAwareEventStore.Append(ctx, streamID, eventstore.VersionCheck(0), event)
	assert.Equal(t, int64(1), newEventStreamVersion)
	assert.NoError(t, err)

	recordedEvents := trackingEventStore.Recorded()
	assert.Len(t, recordedEvents, 1)

	recordedEvent := recordedEvents[0]
	assert.Equal(t, metadata, recordedEvent.Metadata)
}

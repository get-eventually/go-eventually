package correlation_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/extension/correlation"
	"github.com/get-eventually/go-eventually/internal"
)

const randomStringSize = 12

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func TestEventStoreWrapper(t *testing.T) {
	generator := func() string {
		s := make([]rune, randomStringSize)
		for i := range s {
			//nolint:gosec // Weak RNG used for testing only.
			s[i] = alphabet[rand.Intn(len(alphabet))]
		}

		return string(s)
	}

	eventStore := inmemory.NewEventStore()
	correlatedEventStore := correlation.EventStoreWrapper{
		Appender:  eventStore,
		Generator: generator,
	}

	ctx := context.Background()
	streamID := stream.ID{
		Type: "my-type",
		Name: "my-instance",
	}

	metadataWithEventID := eventually.Metadata{}.With(correlation.EventIDKey, "abc123")

	_, err := correlatedEventStore.Append(
		ctx,
		streamID,
		eventstore.VersionCheck(0),
		eventually.Event{Payload: internal.StringPayload("my-first-event")},
		eventually.Event{Payload: internal.StringPayload("my-second-event")},
		eventually.Event{Payload: internal.StringPayload("my-third-event"), Metadata: metadataWithEventID},
	)

	if !assert.NoError(t, err) {
		return
	}

	// Make sure the new events have been recorded with correlation data.
	events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
		return eventStore.Stream(ctx, es, stream.ByID(streamID), eventstore.SelectFromBeginning)
	})

	assert.NoError(t, err)
	assert.Len(t, events, 3)

	firstEvent, secondEvent, thirdEvent := events[0].Event, events[1].Event, events[2].Event

	correlation1, ok := correlation.Message(firstEvent).CorrelationID()
	assert.True(t, ok)

	correlation2, ok := correlation.Message(secondEvent).CorrelationID()
	assert.True(t, ok)

	causation1, ok := correlation.Message(firstEvent).CausationID()
	assert.True(t, ok)

	causation2, ok := correlation.Message(secondEvent).CausationID()
	assert.True(t, ok)

	id1, ok := correlation.Message(firstEvent).EventID()
	assert.True(t, ok)

	id2, ok := correlation.Message(secondEvent).EventID()
	assert.True(t, ok)

	assert.Equal(t, correlation1, correlation2)
	assert.Equal(t, causation1, causation2)
	assert.Equal(t, correlation1, causation1)
	assert.NotEqual(t, id1, id2)

	assert.Equal(t, extractEventID(metadataWithEventID), extractEventID(thirdEvent.Metadata))
}

func extractEventID(m eventually.Metadata) string {
	eventID, _ := m[correlation.EventIDKey].(string)
	return eventID
}

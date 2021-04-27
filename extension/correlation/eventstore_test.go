package correlation_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
	"github.com/eventually-rs/eventually-go/extension/correlation"
	"github.com/eventually-rs/eventually-go/internal"

	"github.com/stretchr/testify/assert"
)

const randomStringSize = 12

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func TestEventStoreWrapper(t *testing.T) {
	generator := func() string {
		s := make([]rune, randomStringSize)
		for i := range s {
			s[i] = alphabet[rand.Intn(len(alphabet))] //nolint:gosec
		}

		return string(s)
	}

	eventStore := eventstore.Store(inmemory.NewEventStore())
	eventStore = correlation.WrapEventStore(eventStore, generator)

	ctx := context.Background()
	typeName := "my-type"
	instanceName := "my-instance"

	// Register stream type and insert a couple of events to inspect later.
	if err := eventStore.Register(ctx, typeName, nil); !assert.NoError(t, err) {
		return
	}

	typedStore, err := eventStore.Type(ctx, typeName)
	if !assert.NoError(t, err) {
		return
	}

	_, err = typedStore.Instance(instanceName).Append(ctx, 0, []eventually.Event{
		{Payload: internal.StringPayload("my-first-event")},
		{Payload: internal.StringPayload("my-second-event")},
	}...)

	if !assert.NoError(t, err) {
		return
	}

	// Make sure the new events have been recorded with correlation data.
	events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
		return typedStore.Stream(ctx, es, 0)
	})

	assert.NoError(t, err)
	assert.Len(t, events, 2)

	firstEvent, secondEvent := events[0].Event, events[1].Event

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
}

package correlation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/event/version"
	"github.com/get-eventually/go-eventually/extension/correlation"
	"github.com/get-eventually/go-eventually/extension/inmemory"
	"github.com/get-eventually/go-eventually/internal"
)

func TestProcessorWrapper(t *testing.T) {
	const (
		typeName      = "my-type"
		correlationID = "fantastic-id"
	)

	ctx := context.Background()
	eventStore := inmemory.NewEventStore()
	correlatedEventStore := correlation.EventStoreWrapper{
		Appender:  eventStore,
		Generator: func() string { return correlationID },
	}

	t.Run("wrapped projector does not extend context if incoming event has no correlation data", func(t *testing.T) {
		streamID := event.StreamID{
			Type: typeName,
			Name: "my-instance",
		}

		_, err := eventStore.Append(
			ctx,
			streamID,
			version.CheckExact(0),
			event.Event{Payload: internal.StringPayload("uncorrelated-test")},
		)

		if !assert.NoError(t, err) {
			return
		}

		events, err := event.StreamToSlice(ctx, func(ctx context.Context, eventStream event.Stream) error {
			return eventStore.Stream(ctx, eventStream, streamID, version.SelectFromBeginning)
		})

		assert.NoError(t, err)
		assert.Len(t, events, 1)

		processor := correlation.ProcessorWrapper{
			Processor: event.ProcessorFunc(func(ctx context.Context, _ event.Persisted) error {
				correlationID, hasCorrelation := correlation.IDContext(ctx)
				causationID, hasCausation := correlation.CausationIDContext(ctx)

				assert.Zero(t, correlationID)
				assert.False(t, hasCorrelation)
				assert.Zero(t, causationID)
				assert.False(t, hasCausation)

				return nil
			}),
		}

		assert.NoError(t, processor.Process(ctx, events[0]))
	})

	t.Run("wrapped projector extends context with incoming event correlation data", func(t *testing.T) {
		streamID := event.StreamID{
			Type: typeName,
			Name: "my-correlated-instance",
		}

		_, err := correlatedEventStore.Append(
			ctx,
			streamID,
			version.CheckExact(0),
			event.Event{Payload: internal.StringPayload("correlated-test")},
		)

		if !assert.NoError(t, err) {
			return
		}

		events, err := event.StreamToSlice(ctx, func(ctx context.Context, eventStream event.Stream) error {
			return eventStore.Stream(ctx, eventStream, streamID, version.SelectFromBeginning)
		})

		assert.NoError(t, err)
		assert.Len(t, events, 1)

		processor := correlation.ProcessorWrapper{
			Processor: event.ProcessorFunc(func(ctx context.Context, _ event.Persisted) error {
				id, hasCorrelation := correlation.IDContext(ctx)
				causationID, hasCausation := correlation.CausationIDContext(ctx)

				assert.Equal(t, correlationID, id)
				assert.Equal(t, correlationID, causationID)
				assert.True(t, hasCorrelation)
				assert.True(t, hasCausation)

				return nil
			}),
		}

		assert.NoError(t, processor.Process(ctx, events[0]))
	})
}

package correlation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/extension/correlation"
	"github.com/get-eventually/go-eventually/internal"
)

type applierFunc func(context.Context, eventstore.Event) error

func (f applierFunc) Apply(ctx context.Context, event eventstore.Event) error { return f(ctx, event) }

func TestProjectionWrapper(t *testing.T) {
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
		streamID := stream.ID{
			Type: typeName,
			Name: "my-instance",
		}

		_, err := eventStore.Append(
			ctx,
			streamID,
			eventstore.VersionCheck(0),
			eventually.Event{Payload: internal.StringPayload("uncorrelated-test")},
		)

		if !assert.NoError(t, err) {
			return
		}

		events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			return eventStore.Stream(ctx, es, stream.ByID(streamID), eventstore.SelectFromBeginning)
		})

		assert.NoError(t, err)
		assert.Len(t, events, 1)

		projection := correlation.ProjectionWrapper{
			Projection: applierFunc(func(ctx context.Context, _ eventstore.Event) error {
				correlationID, hasCorrelation := correlation.IDContext(ctx)
				causationID, hasCausation := correlation.CausationIDContext(ctx)

				assert.Zero(t, correlationID)
				assert.False(t, hasCorrelation)
				assert.Zero(t, causationID)
				assert.False(t, hasCausation)

				return nil
			}),
		}

		assert.NoError(t, projection.Apply(ctx, events[0]))
	})

	t.Run("wrapped projector extends context with incoming event correlation data", func(t *testing.T) {
		streamID := stream.ID{
			Type: typeName,
			Name: "my-correlated-instance",
		}

		_, err := correlatedEventStore.Append(
			ctx,
			streamID,
			eventstore.VersionCheck(0),
			eventually.Event{Payload: internal.StringPayload("correlated-test")},
		)

		if !assert.NoError(t, err) {
			return
		}

		events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			return eventStore.Stream(ctx, es, stream.ByID(streamID), eventstore.SelectFromBeginning)
		})

		assert.NoError(t, err)
		assert.Len(t, events, 1)

		projection := correlation.ProjectionWrapper{
			Projection: applierFunc(func(ctx context.Context, _ eventstore.Event) error {
				id, hasCorrelation := correlation.IDContext(ctx)
				causationID, hasCausation := correlation.CausationIDContext(ctx)

				assert.Equal(t, correlationID, id)
				assert.Equal(t, correlationID, causationID)
				assert.True(t, hasCorrelation)
				assert.True(t, hasCausation)

				return nil
			}),
		}

		assert.NoError(t, projection.Apply(ctx, events[0]))
	})
}

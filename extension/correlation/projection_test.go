package correlation_test

import (
	"context"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
	"github.com/eventually-rs/eventually-go/extension/correlation"
	"github.com/eventually-rs/eventually-go/internal"

	"github.com/stretchr/testify/assert"
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
	correlatedEventStore := correlation.WrapEventStore(eventStore, func() string {
		return correlationID
	})

	if err := eventStore.Register(ctx, typeName, nil); !assert.NoError(t, err) {
		return
	}

	typedStore, err := eventStore.Type(ctx, typeName)
	assert.NoError(t, err)

	correlatedTypedStore, err := correlatedEventStore.Type(ctx, typeName)
	assert.NoError(t, err)

	t.Run("wrapped projector does not extend context if incoming event has no correlation data", func(t *testing.T) {
		const myInstance = "my-instance"

		_, err := typedStore.
			Instance(myInstance).
			Append(ctx, 0, eventually.Event{Payload: internal.StringPayload("uncorrelated-test")})

		if !assert.NoError(t, err) {
			return
		}

		events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			return typedStore.Instance(myInstance).Stream(ctx, es, 0)
		})

		assert.NoError(t, err)
		assert.Len(t, events, 1)

		projection := applierFunc(func(ctx context.Context, _ eventstore.Event) error {
			correlationID, hasCorrelation := correlation.IDContext(ctx)
			causationID, hasCausation := correlation.CausationIDContext(ctx)

			assert.Zero(t, correlationID)
			assert.False(t, hasCorrelation)
			assert.Zero(t, causationID)
			assert.False(t, hasCausation)

			return nil
		})

		assert.NoError(t, correlation.WrapProjection(projection).Apply(ctx, events[0]))
	})

	t.Run("wrapped projector extends context with incoming event correlation data", func(t *testing.T) {
		const myInstance = "my-correlated-instance"

		_, err := correlatedTypedStore.
			Instance(myInstance).
			Append(ctx, 0, eventually.Event{Payload: internal.StringPayload("correlated-test")})

		if !assert.NoError(t, err) {
			return
		}

		events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			return typedStore.Instance(myInstance).Stream(ctx, es, 0)
		})

		assert.NoError(t, err)
		assert.Len(t, events, 1)

		projection := applierFunc(func(ctx context.Context, _ eventstore.Event) error {
			id, hasCorrelation := correlation.IDContext(ctx)
			causationID, hasCausation := correlation.CausationIDContext(ctx)

			assert.Equal(t, correlationID, id)
			assert.Equal(t, correlationID, causationID)
			assert.True(t, hasCorrelation)
			assert.True(t, hasCausation)

			return nil
		})

		assert.NoError(t, correlation.WrapProjection(projection).Apply(ctx, events[0]))
	})
}

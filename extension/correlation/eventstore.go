package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

type Generator func() string

type eventStoreWrapperAccessors struct {
	idGenerator Generator
}

type EventStoreWrapper struct {
	eventstore.Store
	eventStoreWrapperAccessors
}

func WrapEventStore(es eventstore.Store, generator Generator) EventStoreWrapper {
	return EventStoreWrapper{
		Store: es,
		eventStoreWrapperAccessors: eventStoreWrapperAccessors{
			idGenerator: generator,
		},
	}
}

func (es EventStoreWrapper) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	ts, err := es.Store.Type(ctx, typ)
	if err != nil {
		return nil, err
	}

	return typedEventStoreWrapper{
		Typed:                      ts,
		eventStoreWrapperAccessors: es.eventStoreWrapperAccessors,
	}, nil
}

type typedEventStoreWrapper struct {
	eventstore.Typed
	eventStoreWrapperAccessors
}

func (ts typedEventStoreWrapper) Instance(id string) eventstore.Instanced {
	return instancedEventStoreWrapper{
		Instanced:                  ts.Typed.Instance(id),
		eventStoreWrapperAccessors: ts.eventStoreWrapperAccessors,
	}
}

type instancedEventStoreWrapper struct {
	eventstore.Instanced
	eventStoreWrapperAccessors
}

func (is instancedEventStoreWrapper) Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error) {
	// if request id is here, use that; otherwise, build a new id
	causeID := is.idGenerator()

	// Use correlation id from the context.
	// If the context doesn't provide a correlation id, then use the causation id.
	correlationID, ok := ctx.Value(correlationCtxKey{}).(string)
	if !ok || correlationID == "" {
		correlationID = causeID
	}

	causationID, ok := ctx.Value(causationCtxKey{}).(string)
	if !ok || causationID == "" {
		causationID = causeID
	}

	for i, event := range events {
		eventID := is.idGenerator()

		event.Metadata = event.Metadata.
			With(EventIDKey, eventID).
			With(CorrelationIDKey, correlationID).
			With(CausationIDKey, causationID)

		events[i] = event
	}

	return is.Instanced.Append(ctx, version, events...)
}

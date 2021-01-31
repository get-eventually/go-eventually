package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

// Generator is a string identifier generator.
type Generator func() string

type eventStoreWrapperAccessors struct {
	idGenerator Generator
}

// EventStoreWrapper is an extension component that adds support for
// Correlation, Causation and Event id recording in all Events committed
// to the underlying EventStore.
//
// Check WrapEventStore for more information.
type EventStoreWrapper struct {
	eventstore.Store
	eventStoreWrapperAccessors
}

// WrapEventStore wraps the provided eventstore.Store instance with
// an EventStoreWrapper extension.
//
// EventStoreWrapper will add an Event id for each Event committed through
// this instance, using the specified Generator interface.
//
// Also, it will add Correlation and Causation ids in the committed Events
// Metadata, if present in the context. You can check correlation.WithCorrelationID
// and correlation.WithCausationID functions, or correlation.ProjectionWrapper
// for more info.
func WrapEventStore(es eventstore.Store, generator Generator) EventStoreWrapper {
	return EventStoreWrapper{
		Store: es,
		eventStoreWrapperAccessors: eventStoreWrapperAccessors{
			idGenerator: generator,
		},
	}
}

// Type returns an eventstore.Typed instance for the specified stream type identifier,
// with the EventStoreWrapper correlation extension.
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

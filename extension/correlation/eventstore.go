package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

// Generator is a string identifier generator.
type Generator func() string

// EventStoreWrapper is an extension component that adds support for
// Correlation, Causation and Event id recording in all Events committed
// to the underlying EventStore.
//
// Check WrapEventStore for more information.
type EventStoreWrapper struct {
	eventstore.Store
	generateID Generator
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
		Store:      es,
		generateID: generator,
	}
}

func (esw EventStoreWrapper) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	// if request id is here, use that; otherwise, build a new id
	causeID := esw.generateID()

	// Use correlation id from the context.
	// If the context doesn't provide a correlation id, then use the causation id.
	correlationID, ok := IDContext(ctx)
	if !ok || correlationID == "" {
		correlationID = causeID
	}

	causationID, ok := CausationIDContext(ctx)
	if !ok || causationID == "" {
		causationID = causeID
	}

	for i, event := range events {
		eventID := esw.generateID()

		event.Metadata = event.Metadata.
			With(EventIDKey, eventID).
			With(CorrelationIDKey, correlationID).
			With(CausationIDKey, causationID)

		events[i] = event
	}

	return esw.Store.Append(ctx, id, expected, events...)
}

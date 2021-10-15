package correlation

import (
	"context"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

// Generator is a string identifier generator.
type Generator func() string

// EventStoreWrapper is an extension component that adds support for
// Correlation, Causation and Event id recording in all Events committed
// to the underlying EventStore.
//
// EventStoreWrapper will add an Event id for each Event committed through
// this instance, using the specified Generator interface. This is also
// why this wrapper works on eventstore.Appender interface only, since
// it's only extending the Append behavior of the Event Store.
//
// What's more, it will add Correlation and Causation ids in the committed Events
// Metadata, if present in the context. You can check correlation.WithCorrelationID
// and correlation.WithCausationID functions, or correlation.ProjectionWrapper
// for more info.
type EventStoreWrapper struct {
	Appender  eventstore.Appender
	Generator Generator
}

// Append extracts or creates an Event, Correlation and Causation id from the context,
// applies it to all the Events provided and forwards it to the wrapped Event Store.
func (esw EventStoreWrapper) Append(
	ctx context.Context,
	id stream.ID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	// if request id is here, use that; otherwise, build a new id
	causeID := esw.Generator()

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
		eventID, exists := event.Metadata[EventIDKey].(string)
		if !exists {
			eventID = esw.Generator()
		}

		event.Metadata = event.Metadata.
			With(EventIDKey, eventID).
			With(CorrelationIDKey, correlationID).
			With(CausationIDKey, causationID)

		events[i] = event
	}

	return esw.Appender.Append(ctx, id, expected, events...)
}

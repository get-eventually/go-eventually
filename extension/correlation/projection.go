package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/projection"
)

var _ projection.Applier = ProjectionWrapper{}

// ProjectionWrapper is an extension component that adds Correlation
// and Causation ids to the context of the underlying projection.Applier
// instance, if found in the Message received in Apply.
//
// Use WrapProjection to create a new instance.
type ProjectionWrapper struct {
	applier projection.Applier
}

// WrapProjection wraps the specified projection.Applier instance
// with a ProjectionWrapper extension component.
func WrapProjection(applier projection.Applier) ProjectionWrapper {
	return ProjectionWrapper{applier: applier}
}

// Apply applies the provided Event to the wrapped projection.Applier,
// using an augmented context containing Correlation and Causation ids
// in the Event Metadata, if any.
func (pw ProjectionWrapper) Apply(ctx context.Context, event eventstore.Event) error {
	if correlationID, ok := event.Metadata[CorrelationIDKey].(string); ok {
		ctx = WithCorrelationID(ctx, correlationID)
	}

	// New actions that might be taken from inside the Projection are going to be
	// caused by the event applied, hence the usage of "Event-Id" as causation.
	if eventID, ok := event.Metadata[EventIDKey].(string); ok {
		ctx = WithCausationID(ctx, eventID)
	}

	return pw.applier.Apply(ctx, event)
}

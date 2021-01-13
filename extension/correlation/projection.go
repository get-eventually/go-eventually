package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/projection"
)

var _ projection.Applier = ProjectionWrapper{}

type ProjectionWrapper struct {
	applier projection.Applier
}

func WrapProjection(applier projection.Applier) ProjectionWrapper {
	return ProjectionWrapper{applier: applier}
}

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

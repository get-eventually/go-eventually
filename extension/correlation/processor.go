package correlation

import (
	"context"

	"github.com/get-eventually/go-eventually/event"
)

var _ event.Processor = ProcessorWrapper{}

// ProcessorWrapper is an extension component that adds Correlation
// and Causation ids to the context of the underlying event.Processor
// instance, if found in the Message received in Apply.
type ProcessorWrapper struct {
	event.Processor
}

// Apply applies the provided Event to the wrapped projection.Applier,
// using an augmented context containing Correlation and Causation ids
// in the Event Metadata, if any.
func (pw ProcessorWrapper) Process(ctx context.Context, evt event.Persisted) error {
	if correlationID, ok := evt.Metadata[CorrelationIDKey].(string); ok {
		ctx = WithCorrelationID(ctx, correlationID)
	}

	// New actions that might be taken from inside the Projection are going to be
	// caused by the event applied, hence the usage of "Event-Id" as causation.
	if eventID, ok := evt.Metadata[EventIDKey].(string); ok {
		ctx = WithCausationID(ctx, eventID)
	}

	return pw.Processor.Process(ctx, evt)
}

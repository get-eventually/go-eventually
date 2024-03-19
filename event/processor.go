package event

import (
	"context"
)

// Processor represents a component that can process persisted Domain Events.
type Processor interface {
	Process(ctx context.Context, event Persisted) error
}

// ProcessorFunc is a functional implementation of the Processor interface.
type ProcessorFunc func(ctx context.Context, event Persisted) error

// Process implements the event.Processor interface.
func (pf ProcessorFunc) Process(ctx context.Context, event Persisted) error {
	return pf(ctx, event)
}

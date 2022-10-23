package event

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// DefaultRunnerBufferSize is the default size for the buffered channels
// opened by a ProcessorRunner instance, if not specified.
const DefaultRunnerBufferSize = 32

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

// Subscription is used to open an Event Stream from a remote source.
//
// Usually, these are stateful components, and their state can be updated using
// the Checkpoint method.
type Subscription interface {
	Name() string
	Start(ctx context.Context, eventStream StreamWrite) error
	Checkpoint(ctx context.Context, event Persisted) error
}

// ProcessorRunner is an infrastructural component that orchestrates the
// processing of a Domain Event using the provided Event Processor and Subscription,
// to subscribe to incoming events from the Event Store.
type ProcessorRunner struct {
	Processor
	Subscription

	BufferSize int

	Logger func(string, ...any)
}

func (r ProcessorRunner) log(format string, args ...any) {
	if r.Logger == nil {
		return
	}

	r.Logger(format, args...)
}

// Run starts listening to Events from the provided Subscription
// and passing them to the Processor instance for event processing.
//
// Run is a blocking call, that will exit when either the Processor returns an error,
// or the Subscription stops.
//
// Run uses buffered channels to coordinate events communication between components,
// using the value specified in BufferSize, if any, or DefaultRunnerBufferSize otherwise.
//
// To stop the Runner, cancel the provided context.
// If the error returned upon exit is context.Canceled, that usually represent
// a case of normal operation, so it could be treated as a non-error.
func (r ProcessorRunner) Run(ctx context.Context) error {
	if r.BufferSize == 0 {
		r.BufferSize = DefaultRunnerBufferSize
	}

	eventStream := make(chan Persisted, r.BufferSize)
	toCheckpoint := make(chan Persisted, r.BufferSize)

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		r.log("ProcessorRunner started Subscription '%s'", r.Subscription.Name())

		if err := r.Subscription.Start(ctx, eventStream); err != nil {
			return fmt.Errorf("%T: subscription exited with error: %w", r, err)
		}

		return nil
	})

	group.Go(func() error {
		defer close(toCheckpoint)

		for event := range eventStream {
			if err := r.Processor.Process(ctx, event); err != nil {
				return fmt.Errorf("%T: failed to process event: %w", r, err)
			}

			toCheckpoint <- event
		}

		return nil
	})

	for event := range toCheckpoint {
		r.log("Checkpointing processed event, stream id: %s, version: %d", event.StreamID, event.Version)

		if err := r.Subscription.Checkpoint(ctx, event); err != nil {
			return fmt.Errorf("%T: failed to checkpoint processed event: %w", r, err)
		}
	}

	return group.Wait()
}

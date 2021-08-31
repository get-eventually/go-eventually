package projection

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/subscription"
)

// DefaultRunnerBufferSize is the default size for the buffered channels
// opened by a Runner instance, if not specified.
const DefaultRunnerBufferSize = 32

type shouldCheckpointCtxKey struct{}

func contextWithPreferredCheckpointStrategy(ctx context.Context, v *bool) context.Context {
	return context.WithValue(ctx, shouldCheckpointCtxKey{}, v)
}

// Checkpoint should be used in a projection.Apply method to signal
// to a projection.Runner that the event processed should not checkpointed.
//
// NOTE: this is currently the default behavior of projection.Runner, so using
// this method is not usually necessary.
func Checkpoint(ctx context.Context) {
	if v, ok := ctx.Value(shouldCheckpointCtxKey{}).(*bool); ok {
		*v = true
	}
}

// DoNotCheckpoint should be used in a projection.Apply method to signal
// to a projection.Runner that the event processed should not be checkpointed.
func DoNotCheckpoint(ctx context.Context) {
	if v, ok := ctx.Value(shouldCheckpointCtxKey{}).(*bool); ok {
		*v = false
	}
}

// Runner is an infrastructural component that orchestrates the state update
// of an Applier or a Projection using the provided Subscription,
// to subscribe to incoming events from the Event Store.
type Runner struct {
	Applier      Applier
	Subscription subscription.Subscription
	BufferSize   int
}

// Run starts listening to Events from the provided Subscription
// and sinking them to the Applier instance to trigger state change.
//
// Run is a blocking call, that will exit when either the Applier returns an error,
// or the Subscription stops.
//
// Run uses buffered channels to coordinate events communication between components,
// using the value specified in BufferSize, if any, or DefaultRunnerBufferSize otherwise.
//
// To stop the Runner, cancel the provided context.
// If the error returned upon exit is context.Canceled, that usually represent
// a case of normal operation, so it could be treated as a non-error.
func (r Runner) Run(ctx context.Context) error {
	if r.BufferSize == 0 {
		r.BufferSize = DefaultRunnerBufferSize
	}

	eventStream := make(chan eventstore.Event, r.BufferSize)
	toCheckpoint := make(chan eventstore.Event, r.BufferSize)

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		if err := r.Subscription.Start(ctx, eventStream); err != nil {
			return fmt.Errorf("projection.Runner: subscription exited with error: %w", err)
		}

		return nil
	})

	group.Go(func() error {
		defer close(toCheckpoint)

		for event := range eventStream {
			// Default behavior for chechkpointing is to checkpoint every event processed.
			//
			// Users could use projection.DoNotCheckpoint(ctx) to specify whether they want the message
			// not to be checkpointed.
			checkpoint := true
			ctx = contextWithPreferredCheckpointStrategy(ctx, &checkpoint)

			if err := r.Applier.Apply(ctx, event); err != nil {
				return fmt.Errorf("projection.Runner: failed to apply event to projection: %w", err)
			}

			if checkpoint {
				toCheckpoint <- event
			}
		}

		return nil
	})

	for event := range toCheckpoint {
		if err := r.Subscription.Checkpoint(ctx, event); err != nil {
			return fmt.Errorf("projection.Runner: failed to checkpoint event: %w", err)
		}
	}

	return group.Wait()
}

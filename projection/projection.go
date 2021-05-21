package projection

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/query"
	"github.com/get-eventually/go-eventually/subscription"
)

// Applier is a segregated Projection interface that focuses on applying
// Events to a Projection state to trigger a side effect.
//
// Applier implementations do not necessarily require to be Domain Read Models,
// and thus implementing the query.Handler interface.
//
// Examples of pure Appliers are Process Managers, which are long running
// Event listeners that trigger side effects by orchestrating compensating actions.
//
// Use a Runner with a tailored Subscription type to make use
// of an Applier instance.
type Applier interface {
	Apply(context.Context, eventstore.Event) error
}

// ApplierFunc is a functional Applier type implementation.
type ApplierFunc func(ctx context.Context, event eventstore.Event) error

// Apply executes the function pointed by the current value.
func (fn ApplierFunc) Apply(ctx context.Context, event eventstore.Event) error { return fn(ctx, event) }

// Projection represents a Domain Read Model, whose state is updated
// through means of persisted Events coming form the Event Store (the Applier interface),
// and can be accessed using a Domain Query (the query.Handler interface).
//
// Use a Runner with a tailored Subscription type to make use of a Projection.
// Usually, if long-lived and stored in persistent storage, it's suggested to use
// a Catch-up Subscription with a Projection, using a persisted Checkpointer
// to survive application restarts.
type Projection interface {
	query.Handler
	Applier
}

// DefaultRunnerBufferSize is the default size for the buffered channels
// opened by a Runner instance, if not specified.
const DefaultRunnerBufferSize = 32

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
			if err := r.Applier.Apply(ctx, event); err != nil {
				return fmt.Errorf("projection.Runner: failed to apply event to projection: %w", err)
			}

			toCheckpoint <- event
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

package event

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually"
)

// DefaultRunnerBufferSize is the default size for the buffered channels
// opened by a ProcessorRunner instance, if not specified.
const DefaultRunnerBufferSize = 32

type Processor interface {
	Process(ctx context.Context, event Persisted) error
}

type ProcessorFunc func(ctx context.Context, event Persisted) error

func (pf ProcessorFunc) Process(ctx context.Context, event Persisted) error {
	return pf(ctx, event)
}

type Subscription interface {
	Name() string
	Start(ctx context.Context, eventStream Stream) error
	Checkpoint(ctx context.Context, event Persisted) error
}

type shouldCheckpointCtxKey struct{}

func contextWithPreferredCheckpointStrategy(ctx context.Context, v *bool) context.Context {
	return context.WithValue(ctx, shouldCheckpointCtxKey{}, v)
}

// ShouldCheckpoint can be used in a event.Processor.Process method to signal
// to an event.ProcessorRunner that the event processed should not checkpointed.
//
// NOTE: this is currently the default behavior of event.ProcessorRunner, so using
// this method is not usually necessary.
func ShouldCheckpoint(ctx context.Context) {
	if v, ok := ctx.Value(shouldCheckpointCtxKey{}).(*bool); ok {
		*v = true
	}
}

// ShouldNotCheckpoint can be used in a event.Processor.Process method to signal
// to an event.ProcessorRunner that the event processed should not be checkpointed.
func ShouldNotCheckpoint(ctx context.Context) {
	if v, ok := ctx.Value(shouldCheckpointCtxKey{}).(*bool); ok {
		*v = false
	}
}

// ProcessorRunner is an infrastructural component that orchestrates the
// processing of a Domain Event using the provided Event Processor and Subscription,
// to subscribe to incoming events from the Event Store.
type ProcessorRunner struct {
	eventually.Logger

	Processor    Processor
	Subscription Subscription
	BufferSize   int
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
		r.LogInfof(func(log eventually.LoggerFunc) {
			log("ProcessorRunner started Subscription '%s'", r.Subscription.Name())
		})

		if err := r.Subscription.Start(ctx, eventStream); err != nil {
			return errors.Wrap(err, "event.ProcessorRunner.Run: subscription exited with error")
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
			ctx := contextWithPreferredCheckpointStrategy(ctx, &checkpoint) //nolint:govet // Shadowing is fine.

			if err := r.Processor.Process(ctx, event); err != nil {
				return errors.Wrap(err, "event.ProcessorRunner.Run: failed to process event")
			}

			if checkpoint {
				toCheckpoint <- event
				continue
			}

			r.LogDebugf(func(log eventually.LoggerFunc) {
				log("Skip checkpoint of processed event, sequenceNumber: %d", event.SequenceNumber)
			})
		}

		return nil
	})

	for event := range toCheckpoint {
		r.LogDebugf(func(log eventually.LoggerFunc) {
			log("Checkpointing processed event, sequenceNumber: %d", event.SequenceNumber)
		})

		if err := r.Subscription.Checkpoint(ctx, event); err != nil {
			return errors.Wrap(err, "event.ProcessorRunner.Run: failed to checkpoint processed event")
		}
	}

	return group.Wait()
}

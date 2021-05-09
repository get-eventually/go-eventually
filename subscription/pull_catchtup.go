package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/subscription/checkpoint"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/sync/errgroup"
)

// Default values used by a PullCatchUp subscription.
const (
	DefaultPullCatchUpBufferSize = 48
	DefaultPullInterval          = 100 * time.Millisecond
	DefaultMaxPullInterval       = 1 * time.Second
)

var _ Subscription = &PullCatchUp{}

// PullCatchUp represents a catch-up subscription that uses a Streamer
// Event Store to process messages, by "pulling" new Events,
// compared to a CatchUp subscription, which uses a combination of streaming (pulling)
// and subscribing to updates.
type PullCatchUp struct {
	SubscriptionName string
	Target           TargetStream
	EventStore       eventstore.Streamer
	Checkpointer     checkpoint.Checkpointer

	// PullEvery is the minimum interval between each streaming call to the Event Store.
	//
	// Defaults to DefaultPullInterval if unspecified or negative value
	// has been provided.
	PullEvery time.Duration

	// MaxInterval is the maximum interval between each streaming call to the Event Store.
	// Use this value to ensure a specific eventual consistency window.
	//
	// Defaults to DefaultMaxPullInterval if unspecified or negative value
	// has been provided.
	MaxInterval time.Duration

	// BufferSize is the size of buffered channels used as EventStreams
	// by the Subscription when receiving Events from the Event Store.
	//
	// Defaults to DefaultPullCatchUpBufferSize if unspecified or a negative
	// value has been provided.
	BufferSize int
}

// Name is the name of the subscription.
func (s *PullCatchUp) Name() string { return s.SubscriptionName }

// Start starts sending messages on the provided EventStream channel
// by calling the Event Store from where it last left off.
func (s *PullCatchUp) Start(ctx context.Context, stream eventstore.EventStream) error {
	defer close(stream)

	lastSequenceNumber, err := s.Checkpointer.Read(ctx, s.Name())
	if err != nil {
		return fmt.Errorf("subscription.PullCatchUp: failed to read checkpoint: %w", err)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = s.pullEvery()
	b.MaxInterval = s.maxInterval()
	b.MaxElapsedTime = 0 // Don't stop the backoff!

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(b.NextBackOff()):
			sequenceNumber, err := s.catchUp(ctx, stream, lastSequenceNumber)
			if err != nil {
				return fmt.Errorf("subscription.PullCatchUp: failed while streaming: %w", err)
			}

			if sequenceNumber > lastSequenceNumber {
				b.Reset()
			}

			lastSequenceNumber = sequenceNumber
		}
	}
}

func (s *PullCatchUp) catchUp(
	ctx context.Context,
	stream eventstore.EventStream,
	lastSequenceNumber int64,
) (int64, error) {
	es := make(chan eventstore.Event, s.bufferSize())

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		switch t := s.Target.(type) {
		case TargetStreamAll:
			return s.EventStore.StreamAll(ctx, es, eventstore.Select{From: lastSequenceNumber})
		case TargetStreamType:
			return s.EventStore.StreamByType(ctx, es, t.Type, eventstore.Select{From: lastSequenceNumber})
		default:
			return fmt.Errorf("subscription.PullCatchUp: unexpected target type")
		}
	})

	for event := range es {
		stream <- event
		lastSequenceNumber = event.SequenceNumber
	}

	return lastSequenceNumber, group.Wait()
}

func (s *PullCatchUp) pullEvery() time.Duration {
	if s.PullEvery <= 0 {
		return DefaultPullInterval
	}

	return s.PullEvery
}

func (s *PullCatchUp) maxInterval() time.Duration {
	if s.MaxInterval <= 0 {
		return DefaultMaxPullInterval
	}

	return s.MaxInterval
}

func (s *PullCatchUp) bufferSize() int {
	if s.BufferSize <= 0 {
		return DefaultPullCatchUpBufferSize
	}

	return s.BufferSize
}

// Checkpoint uses the Subscription Checkpointer instance to save
// the Global Sequence Number of the Event specified.
func (s *PullCatchUp) Checkpoint(ctx context.Context, event eventstore.Event) error {
	if err := s.Checkpointer.Write(ctx, s.Name(), event.SequenceNumber); err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to checkpoint subscription: %w", err)
	}

	return nil
}

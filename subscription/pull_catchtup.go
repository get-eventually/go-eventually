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

var _ Subscription = PullCatchUp{}

type PullCatchUp struct {
	SubscriptionName string
	Target           TargetStream
	EventStore       eventstore.Streamer
	Checkpointer     checkpoint.Checkpointer

	PullEvery  time.Duration
	MaxTimeout time.Duration
	BufferSize int
}

// Name is the name of the subscription.
func (s PullCatchUp) Name() string { return s.SubscriptionName }

func (s PullCatchUp) Start(ctx context.Context, stream eventstore.EventStream) error {
	defer close(stream)

	lastSequenceNumber, err := s.Checkpointer.Read(ctx, s.Name())
	if err != nil {
		return fmt.Errorf("subscription.PullCatchUp: failed to read checkpoint: %w", err)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = s.PullEvery
	b.MaxInterval = s.MaxTimeout
	b.MaxElapsedTime = 0 // Don't stop the backoff!

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(b.NextBackOff()):
			sequenceNumber, err := s.streamFromStore(ctx, stream, lastSequenceNumber)
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

func (s PullCatchUp) streamFromStore(
	ctx context.Context,
	stream eventstore.EventStream,
	lastSequenceNumber int64,
) (int64, error) {
	es := make(chan eventstore.Event, s.BufferSize)

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

// Checkpoint uses the Subscription Checkpointer instance to save
// the Global Sequence Number of the Event specified.
func (s PullCatchUp) Checkpoint(ctx context.Context, event eventstore.Event) error {
	if err := s.Checkpointer.Write(ctx, s.Name(), event.SequenceNumber); err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to checkpoint subscription: %w", err)
	}

	return nil
}

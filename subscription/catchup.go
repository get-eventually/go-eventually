package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/logger"
	"github.com/get-eventually/go-eventually/subscription/checkpoint"
)

// Default values used by a CatchUp subscription.
const (
	DefaultPullCatchUpBufferSize = 48
	DefaultPullInterval          = 100 * time.Millisecond
	DefaultMaxPullInterval       = 1 * time.Second
)

var _ Subscription = &CatchUp{}

// CatchUp represents a catch-up subscription that uses a Streamer
// Event Store to process messages, by "pulling" new Events,
// compared to a CatchUp subscription, which uses a combination of streaming (pulling)
// and subscribing to updates.
type CatchUp struct {
	SubscriptionName string
	Target           stream.Target
	EventStore       eventstore.Streamer
	Checkpointer     checkpoint.Checkpointer
	Logger           logger.Logger

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
func (s *CatchUp) Name() string { return s.SubscriptionName }

// Start starts sending messages on the provided EventStream channel
// by calling the Event Store from where it last left off.
func (s *CatchUp) Start(ctx context.Context, eventStream eventstore.EventStream) error {
	defer close(eventStream)

	lastSequenceNumber, err := s.Checkpointer.Read(ctx, s.Name())
	if err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to read checkpoint: %w", err)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = s.pullEvery()
	b.MaxInterval = s.maxInterval()
	b.MaxElapsedTime = 0 // Don't stop the backoff!

	logger.Debug(s.Logger, "subscription is starting up",
		logger.With("lastSequenceNumber", lastSequenceNumber),
		logger.With("initialPullInterval", b.InitialInterval),
		logger.With("maxPullInterval", b.MaxInterval),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(b.NextBackOff()):
			sequenceNumber, err := s.catchUp(ctx, eventStream, lastSequenceNumber)
			if err != nil {
				return fmt.Errorf("subscription.CatchUp: failed while streaming: %w", err)
			}

			logger.Debug(s.Logger, "next sequence number recorded",
				logger.With("sequenceNumber", sequenceNumber),
			)

			if sequenceNumber > lastSequenceNumber {
				b.Reset()
			}

			lastSequenceNumber = sequenceNumber
		}
	}
}

func (s *CatchUp) catchUp(
	ctx context.Context,
	eventStream eventstore.EventStream,
	lastSequenceNumber int64,
) (int64, error) {
	es := make(chan eventstore.Event, s.bufferSize())

	// NOTE: incrementing the sequence number by one so that we skip potential
	// duplicates due to inclusive selecting operators.
	selectFrom := eventstore.Select{From: lastSequenceNumber + 1}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return s.EventStore.Stream(ctx, es, s.Target, selectFrom)
	})

	for event := range es {
		logger.Info(s.Logger, "event received",
			logger.With("streamName", event.Stream.Name),
			logger.With("streamType", event.Stream.Type),
			logger.With("version", event.Version),
			logger.With("sequenceNumber", event.SequenceNumber),
		)

		eventStream <- event
		lastSequenceNumber = event.SequenceNumber
	}

	return lastSequenceNumber, group.Wait()
}

func (s *CatchUp) pullEvery() time.Duration {
	if s.PullEvery <= 0 {
		return DefaultPullInterval
	}

	return s.PullEvery
}

func (s *CatchUp) maxInterval() time.Duration {
	if s.MaxInterval <= 0 {
		return DefaultMaxPullInterval
	}

	return s.MaxInterval
}

func (s *CatchUp) bufferSize() int {
	if s.BufferSize <= 0 {
		return DefaultPullCatchUpBufferSize
	}

	return s.BufferSize
}

// Checkpoint uses the Subscription Checkpointer instance to save
// the Global Sequence Number of the Event specified.
func (s *CatchUp) Checkpoint(ctx context.Context, event eventstore.Event) error {
	if err := s.Checkpointer.Write(ctx, s.Name(), event.SequenceNumber); err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to checkpoint subscription: %w", err)
	}

	return nil
}

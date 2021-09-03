package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/logger"
	"github.com/get-eventually/go-eventually/subscription/checkpoint"
)

var _ Subscription = Volatile{}

// Volatile is a Subscription type that does not keep state of
// the last Event processed or received, nor survives the Subscription
// checkpoint between restarts.
//
// Use this Subscription type for volatile processes, such as projecting
// realtime metrics, or when you're only interested in newer events
// committed to the Event Store.
type Volatile struct {
	SubscriptionName string
	Target           stream.Target
	Logger           logger.Logger
	EventStore       interface {
		eventstore.Streamer
		eventstore.SequenceNumberGetter
	}

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
func (v Volatile) Name() string { return v.SubscriptionName }

// Start starts the Subscription by opening a subscribing Event Stream
// using the subscription's Subscriber instance.
func (v Volatile) Start(ctx context.Context, stream eventstore.EventStream) error {
	latestSequenceNumber, err := v.EventStore.LatestSequenceNumber(ctx)
	if err != nil {
		return fmt.Errorf("subscription.Volatile: failed to get latest sequence number from event store: %w", err)
	}

	catchUpSubscription := &CatchUp{
		SubscriptionName: v.SubscriptionName,
		Target:           v.Target,
		EventStore:       v.EventStore,
		Checkpointer:     checkpoint.FixedCheckpointer{StartingFrom: latestSequenceNumber},
		Logger:           v.Logger,
		PullEvery:        v.PullEvery,
		MaxInterval:      v.MaxInterval,
		BufferSize:       v.BufferSize,
	}

	if err := catchUpSubscription.Start(ctx, stream); err != nil {
		return fmt.Errorf("subscription.Volatile: internal catch-up subscription exited with error: %w", err)
	}

	return nil
}

// Checkpoint is a no-op operation, since the transient nature of the
// Subscription does not require to persist its current state.
func (Volatile) Checkpoint(ctx context.Context, event eventstore.Event) error {
	return nil
}

package subscription

import (
	"context"
	"fmt"
	"sync"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/subscription/checkpoint"

	"golang.org/x/sync/errgroup"
)

// DefaultCatchUpBufferSize is the default buffer size of each Event Stream
// used by a CatchUp subscription, when not specified.
const DefaultCatchUpBufferSize = 128

var _ Subscription = CatchUp{}

// CatchUp is a Subscription type that "catches up" with the Event Store
// state, from a certain starting point.
//
// A catch-up subscription uses a Checkpointer to save the latest
// Event processed by the Subscription, and to retrieve such Event reference
// in case of restarts.
//
// A catch-up subscription is perfect for long-running processes, such as
// a Domain Read Model or a Process Manager.
type CatchUp struct {
	SubscriptionName string
	Target           TargetStream
	Checkpointer     checkpoint.Checkpointer
	EventStore       interface {
		eventstore.Streamer
		eventstore.Subscriber
	}

	// BufferSize is the size of buffered channels used as EventStreams
	// by the Subscription when receiving Events from the Event Store.
	//
	// Defaults to DefaultCatchUpBufferSize if unspecified or a negative
	// value has been provided.
	BufferSize int
}

// Name is the name of the subscription.
func (s CatchUp) Name() string { return s.SubscriptionName }

// Start starts the Subscription by opening an Event Stream to catch-up
// with the Event Store state, and establish a volatile Subscription
// to receive new Events notifications from the Event Store.
//
// The latest Event processed checkpoint is retrieved from, and saved to
// using the subscription's provided Checkpointer instance.
//
// Caller of this method will block, unless the provided context gets closed
// or the underlying Event Store calls fail for some reason.
func (s CatchUp) Start(ctx context.Context, stream eventstore.EventStream) error {
	lastSequenceNumber, err := s.Checkpointer.Read(ctx, s.Name())
	if err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to read checkpoint: %w", err)
	}

	defer close(stream)

	streamed := make(chan eventstore.Event, s.bufferSize())
	subscribed := make(chan eventstore.Event, s.bufferSize())

	subscriptionOpened := new(sync.WaitGroup)
	subscriptionOpened.Add(1)

	group, ctx := errgroup.WithContext(ctx)

	// We need specific order that the volatile subscription needs to be open
	// BEFORE the catch-up event stream is opened, so we can catch all events
	// between the start of the Subscription and the last Event Stream state.
	group.Go(func() error {
		subscriptionOpened.Wait()

		switch t := s.Target.(type) {
		case TargetStreamAll:
			return s.EventStore.StreamAll(ctx, streamed, eventstore.Select{From: lastSequenceNumber})
		case TargetStreamType:
			return s.EventStore.StreamByType(ctx, streamed, t.Type, eventstore.Select{From: lastSequenceNumber})
		default:
			return fmt.Errorf("subscription.CatchUp: unexpected target type")
		}
	})

	group.Go(func() error {
		subscriptionOpened.Done()

		switch t := s.Target.(type) {
		case TargetStreamAll:
			return s.EventStore.SubscribeToAll(ctx, subscribed)
		case TargetStreamType:
			return s.EventStore.SubscribeToType(ctx, subscribed, t.Type)
		default:
			return fmt.Errorf("subscription.CatchUp: unexpected target type")
		}
	})

	if err := s.stream(ctx, &lastSequenceNumber, stream, streamed); err != nil {
		return err
	}

	if err := s.stream(ctx, &lastSequenceNumber, stream, subscribed); err != nil {
		return err
	}

	return group.Wait()
}

func (s CatchUp) bufferSize() int {
	if s.BufferSize > 0 {
		return s.BufferSize
	}

	return DefaultCatchUpBufferSize
}

func (s CatchUp) stream(
	ctx context.Context,
	lastSequenceNumber *int64,
	stream eventstore.EventStream,
	source <-chan eventstore.Event,
) error {
	for {
		select {
		case event, ok := <-source:
			if !ok {
				return nil
			}

			if *lastSequenceNumber >= event.SequenceNumber {
				continue
			}

			stream <- event

			// Skip sequence number checkpointing if the event does not have a
			// global sequence number recorded.
			if !ok {
				continue
			}

			*lastSequenceNumber = event.SequenceNumber

		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("subscription.CatchUp: context done: %w", err)
			}

			return nil
		}
	}
}

// Checkpoint uses the Subscription Checkpointer instance to save
// the Global Sequence Number of the Event specified.
func (s CatchUp) Checkpoint(ctx context.Context, event eventstore.Event) error {
	if err := s.Checkpointer.Write(ctx, s.Name(), event.SequenceNumber); err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to checkpoint subscription: %w", err)
	}

	return nil
}

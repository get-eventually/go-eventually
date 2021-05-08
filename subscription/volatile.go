package subscription

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"
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
	Target           TargetStream
	EventStore       eventstore.Subscriber
}

// Name is the name of the subscription.
func (v Volatile) Name() string { return v.SubscriptionName }

// Start starts the Subscription by opening a subscribing Event Stream
// using the subscription's Subscriber instance.
func (v Volatile) Start(ctx context.Context, stream eventstore.EventStream) error {
	var err error

	switch t := v.Target.(type) {
	case TargetStreamAll:
		err = v.EventStore.SubscribeToAll(ctx, stream)
	case TargetStreamType:
		err = v.EventStore.SubscribeToType(ctx, stream, t.Type)
	default:
		return fmt.Errorf("subscription.Volatile: unexpected target type")
	}

	if err != nil {
		return fmt.Errorf("subscription.Volatile: event subscriber exited with error: %w", err)
	}

	return nil
}

// Checkpoint is a no-op operation, since the transient nature of the
// Subscription does not require to persist its current state.
func (Volatile) Checkpoint(ctx context.Context, event eventstore.Event) error {
	return nil
}

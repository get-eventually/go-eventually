package subscription

import (
	"context"

	"github.com/get-eventually/go-eventually/eventstore"
)

// Subscription represents a named subscription to Events coming from
// an Event Stream, usually opened from an Event Store.
//
// Starting a Subscription will open an Event Stream and start sinking
// messages on the provided channel.
//
// Start should be implemented as a synchronous method, which
// should be stopped only when either the underlying Event Store call fails,
// or when the context is explicitly canceled.
//
// Checkpoint should be implemented as a synchronous method, and
// should be called when an event has finished processing successfully.
// Implementations should forward the relevant event data to a Checkpointer
// to persist current state of the Subscription.
type Subscription interface {
	Name() string
	Start(ctx context.Context, eventStream eventstore.EventStream) error
	Checkpoint(ctx context.Context, event eventstore.Event) error
}

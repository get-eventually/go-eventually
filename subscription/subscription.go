package subscription

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
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
type Subscription interface {
	Name() string
	Start(context.Context, eventstore.EventStream) error
}

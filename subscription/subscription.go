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

// TargetStream represents an Event Stream targeted by a Subscription.
//
// Please note, this is a marker interface.
// The only two possible variants are StreamAll and StreamType.
// Use these types instead of creating an instance type.
type TargetStream interface {
	isTargetStream()
}

// TargetStreamAll opens a Subscription that listens to all Events in the Event Store.
type TargetStreamAll struct{}

func (TargetStreamAll) isTargetStream() {
	// NOTE: this is a marker interface implementation.
}

// TargetStreamType opens a Subscription that listens to all Events of a certain
// Stream type, the one specified in the Type field.
type TargetStreamType struct {
	Type string
}

func (TargetStreamType) isTargetStream() {
	// NOTE: this is a marker interface implementation.
}

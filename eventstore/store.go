package eventstore

import (
	"context"

	"github.com/eventually-rs/eventually-go"
)

// Event represents an Event Message that has been persisted into the
// Event Store.
type Event struct {
	eventually.Event

	// StreamType is the type, or category, of the Event Stream to which this
	// Event belong. Usually, this is the name of the Aggregate type.
	StreamType string

	// StreamName is the name of the Event Stream to which this Event belong.
	// Usually, this is the string representation of the Aggregate id.
	StreamName string

	// Version is the version of this Event, used for Optimistic Locking
	// and detecting or avoiding concurrency conflict scenarios.
	Version int64
}

// Streamer is the Event Store trait that deals with opening Event Streams
// from a certain version, or sequence number. Streamer should return
// all the committed Events (after the `from` bound) in the Event Store
// at the time of invocation, instead of opening a long-running subscription
// channel.
//
// Implementations of this interface should be synchronous, returning from
// the call only when all the Events have been streamed into the provided
// Event Stream, and close the channel.
//
// Event Stream channel is provided in input as inversion of dependency,
// in order to allow to callers to choose the desired buffering on the channel,
// matching the caller concurrency properties.
type Streamer interface {
	Stream(ctx context.Context, es EventStream, from int64) error
}

// Subscriber is the Event Store trait that deals with opening a long-running
// subscription channel that receives notifications on newly-committed
// Events into the provided EventStream.
//
// Implementations of this interface should be synchronous, returning from
// the call only when the subscription connection either fails, or the
// subscription gets closed through the provided context.Context.
//
// Event Stream channel is provided in input as inversion of dependency,
// in order to allow to callers to choose the desired buffering on the channel,
// matching the caller concurrency properties.
type Subscriber interface {
	Subscribe(ctx context.Context, es EventStream) error
}

// Store represents an Event Store.
//
// Store gives access to streaming and subscribing to the global Event Store
// Events, which means receiving Events from all the Event Streams committed
// to the Event Store.
type Store interface {
	Streamer
	Subscriber

	// Register registers a new Type identifier, using the provided map of
	// Event identifiers to types needed for deserializing Events from the
	// Event Store to the application.
	Register(ctx context.Context, typ string, events map[string]interface{}) error

	// Type returns access to a Typed Event Store instance related
	// to the type identifier provided.
	//
	// Implementations of this method should check that the type identifier
	// provided has been correctly registered in the Event Store and, if not,
	// return an error if necessary.
	Type(ctx context.Context, typ string) (Typed, error)
}

// Typed represents an Event Store access type, which points to a specific
// Stream type.
//
// A Typed instance is always obtainable from a Store instance, using the
// Type method, specifying the desired Type name.
//
// Stream and Subscribe methods will open Event Streams that will return
// only persisted Events matching the current Stream Type.
type Typed interface {
	Streamer
	Subscriber

	// Instance returns an Instanced access type, which focuses on the specified
	// Event Stream id and Event Stream type represented by the current instance.
	Instance(id string) Instanced
}

// Instanced represents and Event Store access type which focuses on
// a very specific Event Stream.
//
// An Instanced instance is always obtainable from a Typed instance,
// using the Instance method.
type Instanced interface {
	Streamer

	// Append inserts the specified Domain Events into the Event Stream specified
	// by the current instance, returning the new version of the Event Stream.
	//
	// A version can be specified to enable an Optimistic Concurrency check
	// on append, by using the expected version of the Event Stream prior
	// to appending the new Events.
	//
	// Alternatively, -1 can be used if no Optimistic Concurrency check
	// should be carried out.
	Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error)
}

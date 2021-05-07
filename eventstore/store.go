package eventstore

import (
	"context"

	"github.com/eventually-rs/eventually-go"
)

type VersionCheck int64

const VersionCheckAny VersionCheck = iota - 1

type StreamID struct {
	// Type is the type, or category, of the Event Stream to which this
	// Event belong. Usually, this is the name of the Aggregate type.
	Type string

	// Name is the name of the Event Stream to which this Event belong.
	// Usually, this is the string representation of the Aggregate id.
	Name string
}

var SelectFromBeginning = Select{From: 0}

type Select struct {
	From int64
}

// Event represents an Event Message that has been persisted into the
// Event Store.
type Event struct {
	eventually.Event
	StreamID

	// Version is the version of this Event, used for Optimistic Locking
	// and detecting or avoiding concurrency conflict scenarios.
	Version int64
}

// Store represents an Event Store.
//
// Store gives access to streaming and subscribing to the global Event Store
// Events, which means receiving Events from all the Event Streams committed
// to the Event Store.
type Store interface {
	Appender
	Streamer
	Subscriber
}

// Instanced represents and Event Store access type which focuses on
// a very specific Event Stream.
//
// An Instanced instance is always obtainable from a Typed instance,
// using the Instance method.
type Appender interface {
	// Append inserts the specified Domain Events into the Event Stream specified
	// by the current instance, returning the new version of the Event Stream.
	//
	// A version can be specified to enable an Optimistic Concurrency check
	// on append, by using the expected version of the Event Stream prior
	// to appending the new Events.
	//
	// Alternatively, -1 can be used if no Optimistic Concurrency check
	// should be carried out.
	Append(context.Context, StreamID, VersionCheck, ...eventually.Event) (int64, error)
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
	Stream(context.Context, EventStream, StreamID, Select) error
	StreamByType(context.Context, EventStream, string, Select) error
	StreamAll(context.Context, EventStream, Select) error
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
	SubscribeToType(context.Context, EventStream, string) error
	SubscribeToAll(context.Context, EventStream) error
}

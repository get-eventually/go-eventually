package eventstore

import (
	"context"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

// VersionCheck is used to specify the expected version of an Event Stream
// before Append-ing new Events. Useful for optimistic locking.
//
// Use VersionCheckAny if you're not interested in optimistic locking
// and conflict resolution.
type VersionCheck int64

// VersionCheckAny can be used when calling Append() to disregard any
// check on the Event Stream version, when you just want to insert some
// events in the Event Store.
const VersionCheckAny VersionCheck = iota - 1

// Select is used to effectively select a slice of the Event Stream,
// by referencing to either the Event Stream version (in case of Stream)
// or Event Store sequence number (for StreamByType and StreamAll).
type Select struct {
	From int64
}

// SelectFromBeginning is a Select operator instance that will select
// the entirety of the desired Event Stream.
var SelectFromBeginning = Select{From: 0}

// Event represents an Event Message that has been persisted into the
// Event Store.
type Event struct {
	eventually.Event

	// Stream is the identifier of the Event Stream this Event
	// belongs to.
	Stream stream.ID

	// Version is the version of this Event, used for Optimistic Locking
	// and detecting or avoiding concurrency conflict scenarios.
	Version int64

	// Sequence Number is the index of the Event in the Event Store,
	// used for ordered streaming.
	SequenceNumber int64
}

// EventStream is a stream of persisted Events.
type EventStream chan<- Event

// Store represents an Event Store.
type Store interface {
	Appender
	Streamer
}

// Appender is an Event Store trait that provides the ability to append
// new Domain Events to an Event Stream.
//
// Implementations of this interface should be synchronous, returning from
// the call only when either the Events have been correctly saved on the Event Store,
// or if an error occurred.
type Appender interface {
	// Append inserts the specified Domain Events into the Event Stream specified
	// by the current instance, returning the new version of the Event Stream.
	//
	// A version can be specified to enable an Optimistic Concurrency check
	// on append, by using the expected version of the Event Stream prior
	// to appending the new Events.
	//
	// Alternatively, VersionCheckAny can be used if no Optimistic Concurrency check
	// should be carried out.
	//
	// An instance of ErrConflict will be returned if the optimistic locking
	// version check fails against the current version of the Event Stream.
	Append(ctx context.Context, id stream.ID, versionCheck VersionCheck, events ...eventually.Event) (int64, error)
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
	// Stream opens one or more Event Streams as specified by the provided Event Stream target.
	Stream(ctx context.Context, es EventStream, target stream.Target, selectt Select) error
}

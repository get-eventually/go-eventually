package event

import (
	"context"

	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/version"
)

// Stream is a single-use, iterator-backed sequence of persisted Domain Events
// coming from some stream-able source of data, like an Event Store.
//
// Stream is an alias for message.Stream[Persisted]. See [message.Stream] for
// the full iteration and error-reporting contract.
type Stream = message.Stream[Persisted]

// NewStream wraps a producer into a Stream. Convenience re-export of
// [message.NewStream] for values of type [Persisted].
func NewStream(produce func(yield func(Persisted) bool) error) *Stream {
	return message.NewStream(produce)
}

// SliceToStream returns a Stream that yields each element of events in order.
//
// Useful for tests and for adapting fully-buffered results.
func SliceToStream(events []Persisted) *Stream {
	return NewStream(func(yield func(Persisted) bool) error {
		for _, evt := range events {
			if !yield(evt) {
				return nil
			}
		}

		return nil
	})
}

// Streamer is an event.Store trait used to open a specific Event Stream and
// stream it back in the application.
//
// Implementations should respect ctx cancellation between yields by checking
// ctx.Err() at loop boundaries inside the producer.
type Streamer interface {
	Stream(ctx context.Context, id StreamID, selector version.Selector) *Stream
}

// Appender is an event.Store trait used to append new Domain Events in the
// Event Stream.
type Appender interface {
	Append(ctx context.Context, id StreamID, expected version.Check, events ...Envelope) (version.Version, error)
}

// Store represents an Event Store, a stateful data source where Domain Events
// can be safely stored, and easily replayed.
type Store interface {
	Appender
	Streamer
}

// FusedStore is a convenience type to fuse
// multiple Event Store interfaces where you might need to extend
// the functionality of the Store only partially.
//
// E.g. You might want to extend the functionality of the Append() method,
// but keep the Streamer methods the same.
//
// If the extension wrapper does not support
// the Streamer interface, you cannot use the extension wrapper instance as an
// Event Store in certain cases (e.g. the Aggregate Repository).
//
// Using a FusedStore instance you can fuse both instances
// together, and use it with the rest of the library ecosystem.
type FusedStore struct {
	Appender
	Streamer
}

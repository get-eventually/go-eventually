package event

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/core/version"
)

// Stream represents a stream of persisted Domain Events coming from some
// stream-able source of data, like an Event Store.
type Stream[T Event] chan Persisted[T]

// StreamWrite provides write-only access to an event.Stream object.
type StreamWrite[T Event] chan<- Persisted[T]

// StreamRead provides read-only access to an event.Stream object.
type StreamRead[T Event] <-chan Persisted[T]

// SliceToStream converts a slice of event.Persisted domain events to an event.Stream type.
//
// The event.Stream channel has the same buffer size as the input slice.
//
// The channel returned by the function contains all the original slice elements
// and is already closed.
func SliceToStream[T Event](events []Persisted[T]) Stream[T] {
	ch := make(chan Persisted[T], len(events))
	defer close(ch)

	for _, event := range events {
		ch <- event
	}

	return ch
}

// StreamToSlice synchronously exhausts an EventStream to an event.Persisted slice,
// and returns an error if the EventStream origin, passed here as a closure,
// fails with an error.
func StreamToSlice[T Event](ctx context.Context, f func(ctx context.Context, stream StreamWrite[T]) error) ([]Persisted[T], error) {
	ch := make(chan Persisted[T], 1)
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error { return f(ctx, ch) })

	var events []Persisted[T]
	for event := range ch {
		events = append(events, event)
	}

	return events, group.Wait()
}

// Streamer is an event.Store trait used to open a specific Event Stream and stream it back
// in the application.
type Streamer[T Event] interface {
	Stream(ctx context.Context, stream StreamWrite[T], id StreamID, selector version.Selector) error
}

// Appender is an event.Store trait used to append new Domain Events in the Event Stream.
type Appender[T Event] interface {
	Append(ctx context.Context, id StreamID, expected version.Check, events ...Envelope[T]) (version.Version, error)
}

// Store represents an Event Store, a stateful data source where Domain Events
// can be safely stored, and easily replayed.
type Store[T Event] interface {
	Appender[T]
	Streamer[T]
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
type FusedStore[T Event] struct {
	Appender[T]
	Streamer[T]
}

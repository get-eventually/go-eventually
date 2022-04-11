package event

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/core/version"
)

type Stream = chan Persisted

type StreamWrite chan<- Persisted

type StreamRead <-chan Persisted

// StreamToSlice synchronously exhausts an EventStream to an event.Persisted slice,
// and returns an error if the EventStream origin, passed here as a closure,
// fails with an error.
func StreamToSlice(ctx context.Context, f func(ctx context.Context, stream StreamWrite) error) ([]Persisted, error) {
	ch := make(chan Persisted, 1)
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error { return f(ctx, ch) })

	var events []Persisted
	for event := range ch {
		events = append(events, event)
	}

	return events, group.Wait()
}

type Streamer interface {
	Stream(ctx context.Context, stream StreamWrite, id StreamID, selector version.Selector) error
}

type Appender interface {
	Append(ctx context.Context, id StreamID, expected version.Check, events ...Envelope) (version.Version, error)
}

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

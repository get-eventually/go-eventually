package event

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/event/stream"
	"github.com/get-eventually/go-eventually/version"
	"golang.org/x/sync/errgroup"
)

type Stream chan<- Persisted

// StreamToSlice synchronously exhausts an EventStream to an event.Persisted slice,
// and returns an error if the EventStream origin, passed here as a closure,
// fails with an error.
func StreamToSlice(ctx context.Context, f func(ctx context.Context, eventStream Stream) error) ([]Persisted, error) {
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
	Stream(ctx context.Context, eventStream Stream, eventStreamID stream.ID, selector version.Selector) error
}

type Appender interface {
	Append(ctx context.Context, eventStreamID stream.ID, expected version.Check, events ...Event) (uint64, error)
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

// ErrConflict is an error returned by an Event Store when appending
// some events using an expected Event Stream version that does not match
// the current state of the Event Stream.
type ErrConflict struct {
	Expected uint64
	Actual   uint64
}

func (err ErrConflict) Error() string {
	return fmt.Sprintf(
		"event: conflict detected; expected stream version: %d, actual: %d",
		err.Expected,
		err.Actual,
	)
}

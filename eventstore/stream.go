package eventstore

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// EventStream is a stream of persisted Events.
type EventStream chan<- Event

// StreamToSlice synchronously exhausts an EventStream to an Event slice,
// and returns an error if the EventStream origin, passed here as a closure,
// fails with an error.
func StreamToSlice(ctx context.Context, f func(context.Context, EventStream) error) ([]Event, error) {
	ch := make(chan Event, 1)
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error { return f(ctx, ch) })

	var events []Event
	for event := range ch {
		events = append(events, event)
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return events, nil
}

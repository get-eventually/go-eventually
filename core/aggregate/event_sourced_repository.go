package aggregate

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"
)

func RehydrateFromEvents[I ID](root Root[I], eventStream event.StreamRead) error {
	for event := range eventStream {
		if err := root.Apply(event.Message); err != nil {
			return fmt.Errorf("aggregate.RehydrateFromEvents: failed to record event: %w", err)
		}

		root.setVersion(event.Version)
	}

	return nil
}

// Factory is a function that creates new zero-valued
// instances of an aggregate.Root implementation.
type Factory[I ID, T Root[I]] func() T

// EventSourcedRepository provides an aggregate.Repository interface implementation
// that uses an event.Store to store and load the state of the Aggregate Root.
type EventSourcedRepository[I ID, T Root[I]] struct {
	eventStore event.Store
	factory    Factory[I, T]
}

func NewEventSourcedRepository[I ID, T Root[I]](
	eventStore event.Store,
	factory Factory[I, T],
) EventSourcedRepository[I, T] {
	return EventSourcedRepository[I, T]{
		eventStore: eventStore,
		factory:    factory,
	}
}

func (repo EventSourcedRepository[I, T]) Get(ctx context.Context, id I) (T, error) {
	var zeroValue T

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	streamID := event.StreamID(id.String())
	eventStream := make(chan event.Persisted, 1)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := repo.eventStore.Stream(ctx, eventStream, streamID, version.SelectFromBeginning); err != nil {
			return fmt.Errorf("%T: failed while reading event from stream: %w", repo, err)
		}

		return nil
	})

	root := repo.factory()

	if err := RehydrateFromEvents[I](root, eventStream); err != nil {
		return zeroValue, fmt.Errorf("%T: failed to rehydrate aggregate root: %w", repo, err)
	}

	if err := group.Wait(); err != nil {
		return zeroValue, err
	}

	if root.Version() == 0 {
		return zeroValue, ErrRootNotFound
	}

	return root, nil
}

func (repo EventSourcedRepository[I, T]) Save(ctx context.Context, root T) error {
	events := root.FlushRecordedEvents()
	if len(events) == 0 {
		return nil
	}

	streamID := event.StreamID(root.AggregateID().String())
	expectedVersion := version.CheckExact(root.Version() - version.Version(len(events)))

	if _, err := repo.eventStore.Append(ctx, streamID, expectedVersion, events...); err != nil {
		return fmt.Errorf("%T: failed to commit recorded events: %w", repo, err)
	}

	return nil
}

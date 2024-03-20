package aggregate

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

// RehydrateFromEvents rehydrates an Aggregate Root from a read-only Event Stream.
func RehydrateFromEvents[I ID](root Root[I], eventStream event.StreamRead) error {
	for event := range eventStream {
		if err := root.Apply(event.Message); err != nil {
			return fmt.Errorf("aggregate.RehydrateFromEvents: failed to record event: %w", err)
		}

		root.setVersion(event.Version)
	}

	return nil
}

// RehydrateFromState rehydrates an aggregate.Root instance
// using a state type, typically coming from an external state type (e.g. Protobuf type)
// and aggregate.Repository implementation (e.g. postgres.AggregateRepository).
func RehydrateFromState[I ID, Src Root[I], Dst any](
	v version.Version,
	dst Dst,
	deserializer serde.Deserializer[Src, Dst],
) (Src, error) {
	var zeroValue Src

	src, err := deserializer.Deserialize(dst)
	if err != nil {
		return zeroValue, fmt.Errorf("aggregate.RehydrateFromState: failed to deserialize src into dst root, %w", err)
	}

	src.setVersion(v)

	return src, nil
}

// Factory is a function that creates new zero-valued
// instances of an aggregate.Root implementation.
type Factory[I ID, T Root[I]] func() T

// EventSourcedRepository provides an aggregate.Repository interface implementation
// that uses an event.Store to store and load the state of the Aggregate Root.
type EventSourcedRepository[I ID, T Root[I]] struct {
	eventStore event.Store
	typ        Type[I, T]
}

// NewEventSourcedRepository returns a new EventSourcedRepository implementation
// to store and load Aggregate Roots, specified by the aggregate.Type,
// using the provided event.Store implementation.
func NewEventSourcedRepository[I ID, T Root[I]](eventStore event.Store, typ Type[I, T]) EventSourcedRepository[I, T] {
	return EventSourcedRepository[I, T]{
		eventStore: eventStore,
		typ:        typ,
	}
}

// Get returns the Aggregate Root with the specified id.
//
// aggregate.ErrRootNotFound is returned if no Aggregate Root was found with that id.
//
// An error is returned if the underlying Event Store fails, or if an error
// occurs while trying to rehydrate the Aggregate Root state from its Event Stream.
func (repo EventSourcedRepository[I, T]) Get(ctx context.Context, id I) (T, error) {
	var zeroValue T

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	streamID := event.StreamID(id.String())
	eventStream := make(event.Stream, 1)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := repo.eventStore.Stream(ctx, eventStream, streamID, version.SelectFromBeginning); err != nil {
			return fmt.Errorf("aggregate.EventSourcedRepository: failed while reading event from stream, %w", err)
		}

		return nil
	})

	root := repo.typ.Factory()

	if err := RehydrateFromEvents(root, eventStream.AsRead()); err != nil {
		return zeroValue, fmt.Errorf("aggregate.EventSourcedRepository: failed to rehydrate aggregate root, %w", err)
	}

	if err := group.Wait(); err != nil {
		return zeroValue, err
	}

	if root.Version() == 0 {
		return zeroValue, ErrRootNotFound
	}

	return root, nil
}

// Save stores the Aggregate Root to the Event Store, by adding the
// new, uncommitted Domain Events recorded through the Root, if any.
//
// An error is returned if the underlying Event Store fails.
func (repo EventSourcedRepository[I, T]) Save(ctx context.Context, root T) error {
	events := root.FlushRecordedEvents()
	if len(events) == 0 {
		return nil
	}

	streamID := event.StreamID(root.AggregateID().String())
	expectedVersion := version.CheckExact(root.Version() - version.Version(len(events)))

	if _, err := repo.eventStore.Append(ctx, streamID, expectedVersion, events...); err != nil {
		return fmt.Errorf("aggregate.EventSourcedRepository: failed to commit recorded events: %w", err)
	}

	return nil
}

package aggregate

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

// RehydrateFromEvents rehydrates an Aggregate Root from a Stream of persisted
// Domain Events.
//
// The Stream is iterated to completion or until the Aggregate Root's Apply
// method returns an error. After iteration, the stream's terminal error (if
// any) is checked via Stream.Err.
func RehydrateFromEvents[I ID](root Root[I], stream *event.Stream) error {
	for evt := range stream.Iter() {
		if err := root.Apply(evt.Message); err != nil {
			return fmt.Errorf("aggregate.RehydrateFromEvents: failed to record event, %w", err)
		}

		root.setVersion(evt.Version)
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("aggregate.RehydrateFromEvents: streaming failed, %w", err)
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

	streamID := event.StreamID(id.String())
	stream := repo.eventStore.Stream(ctx, streamID, version.SelectFromBeginning)

	root := repo.typ.Factory()
	if err := RehydrateFromEvents(root, stream); err != nil {
		return zeroValue, fmt.Errorf("aggregate.EventSourcedRepository: failed to rehydrate aggregate root, %w", err)
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
	expectedVersion := version.CheckExact(root.Version() - version.Version(len(events))) //nolint:gosec // This should not overflow.

	if _, err := repo.eventStore.Append(ctx, streamID, expectedVersion, events...); err != nil {
		return fmt.Errorf("aggregate.EventSourcedRepository: failed to commit recorded events, %w", err)
	}

	return nil
}

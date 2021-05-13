package aggregate

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/eventstore"

	"golang.org/x/sync/errgroup"
)

// ErrRootNotFound is returned by the Repository when no Events for the
// specified Aggregate Root have been found.
var ErrRootNotFound = fmt.Errorf("aggregate.Repository: aggregate root not found")

// RepositoryEventStore is the Event Store interface used by the Repository.
type RepositoryEventStore interface {
	eventstore.Appender
	eventstore.Streamer
}

// Repository is an Event-sourced Repository implementation for retrieving
// and saving Aggregate Roots, using an underlying Event Store instance.
//
// Use `NewRepository` to create a new instance of this type.
type Repository struct {
	eventStore    RepositoryEventStore
	aggregateType Type
}

// NewRepository creates a new instance for retrieving and saving Aggregate Roots
// of the Aggregate type specified in the supplied Type argument.
func NewRepository(aggregateType Type, eventStore RepositoryEventStore) *Repository {
	return &Repository{aggregateType: aggregateType, eventStore: eventStore}
}

// Add saves the provided Aggregate Root instance into the Event Store,
// by flushing any uncommitted event produced while handling commands.
//
// An error is returned if the underlying Event Store fails to commit the
// Aggregate's events.
func (r *Repository) Add(ctx context.Context, root Root) error {
	events := root.flushRecordedEvents()
	if len(events) == 0 {
		return nil
	}

	streamID := eventstore.StreamID{
		Type: r.aggregateType.Name(),
		Name: root.AggregateID().String(),
	}

	expectedVersion := eventstore.VersionCheck(root.Version() - int64(len(events)))

	_, err := r.eventStore.Append(ctx, streamID, expectedVersion, events...)
	if err != nil {
		return fmt.Errorf("aggregate.Repository: failed to commit recorded events: %w", err)
	}

	return nil
}

// Get retrieves the Aggregate Root instance identified by the provided ID.
//
// ErrRootNotFound is returned if no Events for the specified Aggregate Root
// have been found.
//
// An error is also returned if the underlying Event Store fails to
// stream the Root's Event Stream back into the application.
func (r Repository) Get(ctx context.Context, id ID) (Root, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	root, isEmpty := r.aggregateType.instance(), true
	stream := make(chan eventstore.Event, 1)

	streamID := eventstore.StreamID{
		Type: r.aggregateType.Name(),
		Name: id.String(),
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := r.eventStore.Stream(ctx, stream, streamID, eventstore.SelectFromBeginning); err != nil {
			return fmt.Errorf("aggregate.Repository: failed while reading event from stream: %w", err)
		}

		return nil
	})

	for event := range stream {
		isEmpty = false

		if err := root.Apply(event.Event); err != nil {
			return nil, fmt.Errorf("aggregate.Repository: failed to apply event while rehydrating aggregate: %w", err)
		}

		root.updateVersion(event.Version)
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	if isEmpty {
		return nil, ErrRootNotFound
	}

	return root, nil
}

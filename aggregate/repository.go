package aggregate

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"

	"golang.org/x/sync/errgroup"
)

// ErrRootNotFound is returned by the Repository when no Events for the
// specified Aggregate Root have been found.
var ErrRootNotFound = fmt.Errorf("aggregate.Repository: aggregate root not found")

// Repository is an Event-sourced Repository implementation for retrieving
// and saving Aggregate Roots, using an underlying Event Store instance.
//
// Use `NewRepository` to create a new instance of this type.
type Repository struct {
	eventStore    eventstore.Typed
	aggregateType Type
}

// NewRepository creates a new instance for retrieving and saving Aggregate Roots
// of the Aggregate type specified in the supplied Type argument.
func NewRepository(aggregateType Type, eventStore eventstore.Typed) *Repository {
	return &Repository{aggregateType: aggregateType, eventStore: eventStore}
}

// Add saves the provided Aggregate Root instance into the Event Store,
// by flushing any uncommitted event produced while handling commands.
//
// An error is returned if the underlying Event Store fails to commit the
// Aggregate's events.
func (r *Repository) Add(ctx context.Context, root Root) error {
	events := root.flushRecordedEvents()
	newVersion, err := r.eventStore.
		Instance(root.AggregateID().String()).
		Append(ctx, root.Version(), events...)

	if err != nil {
		return fmt.Errorf("aggregate.Repository: failed to commit recorded events: %w", err)
	}

	root.updateVersion(newVersion)

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

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := r.eventStore.Instance(id.String()).Stream(ctx, stream, 0); err != nil {
			return fmt.Errorf("aggregate.Repository: failed while reading event from stream: %w", err)
		}

		return nil
	})

	for event := range stream {
		if err := root.Apply(event.Event); err != nil {
			return nil, fmt.Errorf("aggregate.Repository: failed to apply event while rehydrating aggregate: %w", err)
		}

		isEmpty = false

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

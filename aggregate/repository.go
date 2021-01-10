package aggregate

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"

	"golang.org/x/sync/errgroup"
)

var ErrRootNotFound = fmt.Errorf("aggregate.Repository: aggregate root not found")

type Repository struct {
	eventStore    eventstore.Typed
	aggregateType Type
}

func NewRepository(aggregateType Type, eventStore eventstore.Typed) *Repository {
	return &Repository{aggregateType: aggregateType, eventStore: eventStore}
}

func (r *Repository) Add(ctx context.Context, root Root) error {
	events := root.flushRecordedEvents()
	newVersion, err := r.eventStore.Instance(root.AggregateID()).Append(ctx, root.Version(), events...)

	if err != nil {
		return fmt.Errorf("aggregate.Repository: failed to commit recorded events: %w", err)
	}

	root.updateVersion(newVersion)
	return nil
}

func (r Repository) Get(ctx context.Context, id string) (Root, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	root, isEmpty := r.aggregateType.instance(), true
	stream := make(chan eventstore.Event, 1)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := r.eventStore.Instance(id).Stream(ctx, stream, 0); err != nil {
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

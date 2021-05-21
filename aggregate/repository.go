package aggregate

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually/aggregate/snapshot"
	"github.com/get-eventually/go-eventually/eventstore"
)

// ErrRootNotFound is returned by the Repository when no Events for the
// specified Aggregate Root have been found.
var ErrRootNotFound = fmt.Errorf("aggregate.Repository: aggregate root not found")

// RepositoryEventStore is the Event Store interface used by the Repository.
type RepositoryEventStore interface {
	eventstore.Appender
	eventstore.Streamer
}

// RepositorySnapshotStore is the Snapshot Store interface used by the Repository.
type RepositorySnapshotStore interface {
	snapshot.Getter
	snapshot.Recorder
}

// RepositoryOption is an optional argument to build a Repository interface
// to extend its default functionality.
type RepositoryOption interface {
	apply(*Repository)
}

type repositoryOption func(*Repository)

func (fn repositoryOption) apply(r *Repository) { fn(r) }

// RepositoryWithSnapshotStore uses the provided Snapshot store instance to retrieve
// and save periodical Aggregate Root snapshots, following the Snapshot Policy
// provided to the Repository.
//
// Remember to set a proper Snapshot Policy to make use of the Store,
// as the default behavior from the Repository is to never perform one.
func RepositoryWithSnapshotStore(store RepositorySnapshotStore) RepositoryOption {
	return repositoryOption(func(r *Repository) {
		r.snapshotStore = store
	})
}

// RepositoryWithSnapshotPolicy uses the provided Snapshot policy to decide
// whether to take a snapshot of the state, when Add() is called.
//
// The default policy used by the Repository is snapshot.NeverPolicy,
// which never writes anything to the Snapshot store.
//
// Remember to provide a non-nil Snapshot store isntance using RepositoryWithSnapshotStore,
// otherwise the Repository will skip the snapshotting logic.
func RepositoryWithSnapshotPolicy(policy snapshot.Policy) RepositoryOption {
	return repositoryOption(func(r *Repository) {
		if r.snapshotPolicy != nil {
			r.snapshotPolicy = policy
		}
	})
}

// Repository is an Event-sourced Repository implementation for retrieving
// and saving Aggregate Roots, using an underlying Event Store instance.
//
// Use `NewRepository` to create a new instance of this type.
type Repository struct {
	aggregateType  Type
	eventStore     RepositoryEventStore
	snapshotStore  RepositorySnapshotStore
	snapshotPolicy snapshot.Policy
}

// NewRepository creates a new instance for retrieving and saving Aggregate Roots
// of the Aggregate type specified in the supplied Type argument.
func NewRepository(aggregateType Type, eventStore RepositoryEventStore, options ...RepositoryOption) *Repository {
	r := &Repository{
		aggregateType:  aggregateType,
		eventStore:     eventStore,
		snapshotPolicy: snapshot.NeverPolicy{},
	}

	for _, option := range options {
		if option != nil {
			option.apply(r)
		}
	}

	return r
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

	if err := r.maybeTakeASnapshot(ctx, root); err != nil {
		return fmt.Errorf("aggregate.Repository: failed to take a snapshot of the aggregate state: %w", err)
	}

	return nil
}

func (r *Repository) maybeTakeASnapshot(ctx context.Context, root Root) error {
	if r.snapshotStore == nil || !r.snapshotPolicy.ShouldRecord(root.Version()) {
		return nil
	}

	err := r.snapshotStore.Record(ctx, root.AggregateID().String(), root.Version(), root)
	if err != nil {
		return err
	}

	r.snapshotPolicy.Record(root.Version())

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

	streamID := eventstore.StreamID{
		Type: r.aggregateType.Name(),
		Name: id.String(),
	}

	root, isEmpty, err := r.fetchAggregateRootState(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("aggregate.Repository: failed to fetch initial aggregate root state: %w", err)
	}

	stream := make(chan eventstore.Event, 1)
	streamSelect := eventstore.Select{From: root.Version()}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := r.eventStore.Stream(ctx, stream, streamID, streamSelect); err != nil {
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

func (r Repository) fetchAggregateRootState(ctx context.Context, id eventstore.StreamID) (Root, bool, error) {
	state := r.aggregateType.instance()
	isEmpty := true

	if r.snapshotStore == nil {
		return state, isEmpty, nil
	}

	snapshotState, err := r.snapshotStore.Get(ctx, id.Name)
	if errors.Is(err, snapshot.ErrNotFound) {
		return state, isEmpty, nil
	} else if err != nil {
		return nil, false, err
	}

	rootState, ok := snapshotState.State.(Root)
	if !ok {
		// NOTE(ar3s3ru): this is an important failure, we might want to log it.
		// However, failing the whole operation because of it is not ideal.
		return state, isEmpty, nil
	}

	// This is to prevent the Repository from returning ErrNotFound.
	isEmpty = false

	rootState.updateVersion(snapshotState.Version)

	return rootState, isEmpty, nil
}

package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/aggregate/snapshot"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/extension/inmemory"
)

// Aggregate definition, used in the tests -------------------------------------

var AggregateType = aggregate.NewType("aggregate", func() aggregate.Root {
	return new(Aggregate)
})

type Aggregate struct {
	aggregate.BaseRoot

	ID             aggregate.StringID `json:"id"`
	EventsRecorded uint               `json:"events_recorded"`
}

func (a Aggregate) AggregateID() aggregate.ID { return a.ID }

type AggregateCreated struct {
	AggregateID string
}

func (AggregateCreated) Name() string { return "aggregate_created" }

type AggregateEventRecorded struct {
	AggregateID string
}

func (AggregateEventRecorded) Name() string { return "aggregate_event_recorded" }

func (a *Aggregate) Apply(evt event.Event) error {
	switch event := evt.Payload.(type) {
	case AggregateCreated:
		a.ID = aggregate.StringID(event.AggregateID)
	case AggregateEventRecorded:
		a.EventsRecorded++
	default:
		return fmt.Errorf("aggregate_test.Aggregate: invalid event type")
	}

	return nil
}

func NewAggregate(id string) (*Aggregate, error) {
	var a Aggregate

	if err := aggregate.RecordThat(&a, event.Event{
		Payload: AggregateCreated{AggregateID: id},
	}); err != nil {
		return nil, fmt.Errorf("aggregate_test.Aggrgate: failed to record event: %w", err)
	}

	return &a, nil
}

func (a *Aggregate) RecordEvent() error {
	return aggregate.RecordThat(a, event.Event{
		Payload: AggregateEventRecorded{AggregateID: a.ID.String()},
	})
}

// -----------------------------------------------------------------------------

func TestRepository(t *testing.T) {
	ctx := context.Background()
	eventStore := inmemory.NewEventStore()
	snapshotStore := snapshot.NewInMemoryStore()
	repository := aggregate.NewRepository(AggregateType, eventStore,
		aggregate.RepositoryWithSnapshotPolicy(snapshot.AlwaysPolicy{}),
		aggregate.RepositoryWithSnapshotStore(snapshotStore),
	)

	t.Run("no aggregate root found if no event has been recorded", func(t *testing.T) {
		id := aggregate.StringID("test-aggregate-not-found")

		root, err := repository.Get(ctx, id)
		assert.Nil(t, root)
		assert.True(t, errors.Is(err, aggregate.ErrRootNotFound), "error", err.Error())

		state, err := snapshotStore.Get(ctx, id.String())
		assert.Zero(t, state)
		assert.ErrorIs(t, err, snapshot.ErrNotFound)
	})

	t.Run("creating and adding an aggregate works", func(t *testing.T) {
		aggregateID := aggregate.StringID("test-new-aggregate")
		root, err := NewAggregate(aggregateID.String())
		assert.NoError(t, err)

		assert.NoError(t, root.RecordEvent())

		assert.NoError(t, repository.Add(ctx, root))

		againRoot, err := repository.Get(ctx, aggregateID)
		assert.NoError(t, err)
		assert.Equal(t, root, againRoot)

		snapshotRoot, err := snapshotStore.Get(ctx, aggregateID.String())
		assert.NoError(t, err)
		assert.Equal(t, root.Version(), snapshotRoot.Version)
		assert.Equal(t, root, snapshotRoot.State)
	})
}

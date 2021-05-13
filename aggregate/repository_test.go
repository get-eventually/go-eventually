package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"

	"github.com/stretchr/testify/assert"
)

// Aggregate definition, used in the tests -------------------------------------

var AggregateType = aggregate.NewType("aggregate", func() aggregate.Root {
	return new(Aggregate)
})

type Aggregate struct {
	aggregate.BaseRoot

	id             aggregate.StringID
	eventsRecorded uint
}

func (a Aggregate) AggregateID() aggregate.ID { return a.id }

type AggregateCreated struct {
	AggregateID string
}

func (AggregateCreated) Name() string { return "aggregate_created" }

type AggregateEventRecorded struct {
	AggregateID string
}

func (AggregateEventRecorded) Name() string { return "aggregate_event_recorded" }

func (a *Aggregate) Apply(evt eventually.Event) error {
	switch event := evt.Payload.(type) {
	case AggregateCreated:
		a.id = aggregate.StringID(event.AggregateID)
	case AggregateEventRecorded:
		a.eventsRecorded++
	default:
		return fmt.Errorf("aggregate_test.Aggregate: invalid event type")
	}

	return nil
}

func NewAggregate(id string) (*Aggregate, error) {
	var a Aggregate

	err := aggregate.RecordThat(&a, eventually.Event{
		Payload: AggregateCreated{AggregateID: id},
	})
	if err != nil {
		return nil, fmt.Errorf("aggregate_test.Aggrgate: failed to record event: %w", err)
	}

	return &a, nil
}

func (a *Aggregate) RecordEvent() error {
	return aggregate.RecordThat(a, eventually.Event{
		Payload: AggregateEventRecorded{AggregateID: a.id.String()},
	})
}

// -----------------------------------------------------------------------------

func TestRepository(t *testing.T) {
	ctx := context.Background()
	eventStore := inmemory.NewEventStore()
	repository := aggregate.NewRepository(AggregateType, eventStore)

	t.Run("no aggregate root found if no event has been recorded", func(t *testing.T) {
		root, err := repository.Get(ctx, aggregate.StringID("test-aggregate-not-found"))
		assert.Nil(t, root)
		assert.True(t, errors.Is(err, aggregate.ErrRootNotFound), "error", err.Error())
	})

	t.Run("creating and adding an aggregate works", func(t *testing.T) {
		aggregateID := aggregate.StringID("test-new-aggregate")
		root, err := NewAggregate(aggregateID.String())
		assert.NoError(t, err)

		err = repository.Add(ctx, root)
		assert.NoError(t, err)

		againRoot, err := repository.Get(ctx, aggregateID)
		assert.NoError(t, err)
		assert.Equal(t, root, againRoot)
	})
}

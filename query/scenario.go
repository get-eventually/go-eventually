package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/version"
)

// ProcessorHandler is a Query Handler that can both handle domain queries,
// and domain events to hydrate the query model.
//
// To be used in the Scenario.
type ProcessorHandler[Q Query, R any] interface {
	Handler[Q, R]
	event.Processor
}

// ScenarioInit is the entrypoint of the Query Handler scenario API.
//
// A Query Handler scenario can either set the current evaluation context
// by using Given(), or test a "clean-slate" scenario by using When() directly.
type ScenarioInit[Q Query, R any, T ProcessorHandler[Q, R]] struct{}

// Scenario can be used to test the result of Domain Queries
// being handled by a Query Handler.
//
// Query Handlers in Event-sourced systems return read-only data on request by means
// of Domain Queries. This scenario API helps you with testing the values
// returned by a Query Handler when handling a specific Domain Query.
func Scenario[Q Query, R any, T ProcessorHandler[Q, R]]() ScenarioInit[Q, R, T] {
	return ScenarioInit[Q, R, T]{}
}

// Given sets the Query Handler scenario preconditions.
//
// Domain Events are used in Event-sourced systems to represent a side effect
// that has taken place in the system. In order to set a given state for the
// system to be in while testing a specific Domain Query evaluation, you should
// specify the Domain Events that have happened thus far.
//
// When you're testing Domain Queries with a clean-slate system, you should either specify
// no Domain Events, or skip directly to When().
func (sc ScenarioInit[Q, R, T]) Given(events ...event.Persisted) ScenarioGiven[Q, R, T] {
	return ScenarioGiven[Q, R, T]{
		given: events,
	}
}

// When provides the Domain Query to evaluate.
func (sc ScenarioInit[Q, R, T]) When(q Envelope[Q]) ScenarioWhen[Q, R, T] {
	//nolint:exhaustruct // Zero values are fine here.
	return ScenarioWhen[Q, R, T]{
		when: q,
	}
}

// ScenarioGiven is the state of the scenario once
// a set of Domain Events have been provided using Given(), to represent
// the state of the system at the time of evaluating a Domain Event.
type ScenarioGiven[Q Query, R any, T ProcessorHandler[Q, R]] struct {
	given []event.Persisted
}

// When provides the Command to evaluate.
func (sc ScenarioGiven[Q, R, T]) When(q Envelope[Q]) ScenarioWhen[Q, R, T] {
	return ScenarioWhen[Q, R, T]{
		ScenarioGiven: sc,
		when:          q,
	}
}

// ScenarioWhen is the state of the scenario once the state of the
// system and the Domain Query to evaluate has been provided.
type ScenarioWhen[Q Query, R any, T ProcessorHandler[Q, R]] struct {
	ScenarioGiven[Q, R, T]
	when Envelope[Q]
}

// Then sets a positive expectation on the scenario outcome, to produce
// the Query Result provided in input.
func (sc ScenarioWhen[Q, R, T]) Then(result R) ScenarioThen[Q, R, T] {
	//nolint:exhaustruct // Zero values are fine here.
	return ScenarioThen[Q, R, T]{
		ScenarioWhen: sc,
		then:         result,
	}
}

// ThenError sets a negative expectation on the scenario outcome,
// to produce an error value that is similar to the one provided in input.
//
// Error assertion happens using errors.Is(), so the error returned
// by the Query Handler is unwrapped until the cause error to match
// the provided expectation.
func (sc ScenarioWhen[Q, R, T]) ThenError(err error) ScenarioThen[Q, R, T] {
	//nolint:exhaustruct // Zero values are fine here.
	return ScenarioThen[Q, R, T]{
		ScenarioWhen: sc,
		wantError:    true,
		thenError:    err,
	}
}

// ThenFails sets a negative expectation on the scenario outcome,
// to fail the Domain Query evaluation with no particular assertion on the error returned.
//
// This is useful when the error returned is not important for the Domain Query
// you're trying to test.
func (sc ScenarioWhen[Q, R, T]) ThenFails() ScenarioThen[Q, R, T] {
	//nolint:exhaustruct // Zero values are fine here.
	return ScenarioThen[Q, R, T]{
		ScenarioWhen: sc,
		wantError:    true,
	}
}

// ScenarioThen is the state of the scenario once the preconditions
// and expectations have been fully specified.
type ScenarioThen[Q Query, R any, T ProcessorHandler[Q, R]] struct {
	ScenarioWhen[Q, R, T]

	then      R
	thenError error
	wantError bool
}

// AssertOn performs the specified expectations of the scenario, using the Query Handler
// instance produced by the provided factory function.
func (sc ScenarioThen[Q, R, T]) AssertOn( //nolint:gocritic
	t *testing.T,
	handlerFactory func(es event.Store) T,
) {
	ctx := context.Background()

	eventStore := event.NewInMemoryStore()
	queryHandler := handlerFactory(eventStore)

	for _, evt := range sc.given {
		_, err := eventStore.Append(ctx, evt.StreamID, version.CheckExact(evt.Version-1), evt.Envelope)
		require.NoError(t, err, "failed to record event on the event store", evt)

		err = queryHandler.Process(ctx, evt)
		require.NoError(t, err, "event failed to be processed with the query handler", evt)
	}

	actual, err := queryHandler.Handle(ctx, sc.when)

	if !sc.wantError {
		assert.NoError(t, err)
		assert.Equal(t, sc.then, actual)

		return
	}

	if !assert.Error(t, err) {
		return
	}

	if sc.thenError != nil {
		assert.ErrorIs(t, err, sc.thenError)
	}
}

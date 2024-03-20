package command

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/version"
)

// ScenarioInit is the entrypoint of the Command Handler scenario API.
//
// A Command Handler scenario can either set the current evaluation context
// by using Given(), or test a "clean-slate" scenario by using When() directly.
type ScenarioInit[Cmd Command, T Handler[Cmd]] struct{}

// Scenario is a scenario type to test the result of Commands
// being handled by a Command Handler.
//
// Command Handlers in Event-sourced systems produce side effects by means
// of Domain Events. This scenario API helps you with testing the Domain Events
// produced by a Command Handler when handling a specific Command.
func Scenario[Cmd Command, T Handler[Cmd]]() ScenarioInit[Cmd, T] {
	return ScenarioInit[Cmd, T]{}
}

// Given sets the Command Handler scenario preconditions.
//
// Domain Events are used in Event-sourced systems to represent a side effect
// that has taken place in the system. In order to set a given state for the
// system to be in while testing a specific Command evaluation, you should
// specify the Domain Events that have happened thus far.
//
// When you're testing Commands with a clean-slate system, you should either specify
// no Domain Events, or skip directly to When().
func (sc ScenarioInit[Cmd, T]) Given(events ...event.Persisted) ScenarioGiven[Cmd, T] {
	return ScenarioGiven[Cmd, T]{
		given: events,
	}
}

// When provides the Command to evaluate.
func (sc ScenarioInit[Cmd, T]) When(cmd Envelope[Cmd]) ScenarioWhen[Cmd, T] {
	return ScenarioWhen[Cmd, T]{
		ScenarioGiven: ScenarioGiven[Cmd, T]{given: nil},
		when:          cmd,
	}
}

// ScenarioGiven is the state of the scenario once
// a set of Domain Events have been provided using Given(), to represent
// the state of the system at the time of evaluating a Command.
type ScenarioGiven[Cmd Command, T Handler[Cmd]] struct {
	given []event.Persisted
}

// When provides the Command to evaluate.
func (sc ScenarioGiven[Cmd, T]) When(cmd Envelope[Cmd]) ScenarioWhen[Cmd, T] {
	return ScenarioWhen[Cmd, T]{
		ScenarioGiven: sc,
		when:          cmd,
	}
}

// ScenarioWhen is the state of the scenario once the state of the
// system and the Command to evaluate have been provided.
type ScenarioWhen[Cmd Command, T Handler[Cmd]] struct {
	ScenarioGiven[Cmd, T]

	when Envelope[Cmd]
}

// Then sets a positive expectation on the scenario outcome, to produce
// the Domain Events provided in input.
//
// The list of Domain Events specified should be ordered as the expected
// order of recording by the Command Handler.
func (sc ScenarioWhen[Cmd, T]) Then(events ...event.Persisted) ScenarioThen[Cmd, T] {
	return ScenarioThen[Cmd, T]{
		ScenarioWhen: sc,
		then:         events,
		errors:       nil,
		wantErr:      false,
	}
}

// ThenError sets a negative expectation on the scenario outcome,
// to produce an error value that is similar to the one provided in input.
//
// Error assertion happens using errors.Is(), so the error returned
// by the Command Handler is unwrapped until the cause error to match
// the provided expectation.
func (sc ScenarioWhen[Cmd, T]) ThenError(err error) ScenarioThen[Cmd, T] {
	return ScenarioThen[Cmd, T]{
		ScenarioWhen: sc,
		then:         nil,
		errors:       []error{err},
		wantErr:      true,
	}
}

// ThenErrors specifies an unsuccessful outcome of the scenario, where the domain command
// execution fails with a specific error that wraps multiple error types (e.g. through `errors.Join`).
//
// Use this method when you want to assert that the error returned by the domain command
// matches ALL of the errors specified.
func (sc ScenarioWhen[I, T]) ThenErrors(errs ...error) ScenarioThen[I, T] {
	return ScenarioThen[I, T]{
		ScenarioWhen: sc,
		then:         nil,
		errors:       errs,
		wantErr:      true,
	}
}

// ThenFails sets a negative expectation on the scenario outcome,
// to fail the Command execution with no particular assertion on the error returned.
//
// This is useful when the error returned is not important for the Command
// you're trying to test.
func (sc ScenarioWhen[Cmd, T]) ThenFails() ScenarioThen[Cmd, T] {
	return ScenarioThen[Cmd, T]{
		ScenarioWhen: sc,
		then:         nil,
		errors:       nil,
		wantErr:      true,
	}
}

// ScenarioThen is the state of the scenario once the preconditions
// and expectations have been fully specified.
type ScenarioThen[Cmd Command, T Handler[Cmd]] struct {
	ScenarioWhen[Cmd, T]

	then    []event.Persisted
	errors  []error
	wantErr bool
}

// AssertOn performs the specified expectations of the scenario, using the Command Handler
// instance produced by the provided factory function.
//
// A Command Handler should only use a single Aggregate type, to ensure that the
// side effects happen in a well-defined transactional boundary. If your Command Handler
// needs to modify more than one Aggregate, you might be doing something wrong
// in your domain model.
//
// The type of the Aggregate used to evaluate the Command must be specified,
// so that the Event-sourced Repository instance can be provided to the factory function
// to build the desired Command Handler.
func (sc ScenarioThen[Cmd, T]) AssertOn(
	t *testing.T,
	handlerFactory func(event.Store) T,
) {
	ctx := context.Background()
	store := event.NewInMemoryStore()

	for _, evt := range sc.given {
		_, err := store.Append(ctx, evt.StreamID, version.Any, evt.Envelope)
		require.NoError(t, err)
	}

	trackingStore := event.NewTrackingStore(store)
	handler := handlerFactory(event.FusedStore{
		Appender: trackingStore,
		Streamer: store,
	})

	switch err := handler.Handle(context.Background(), sc.when); {
	case sc.wantErr:
		assert.Error(t, err)

		if expected := errors.Join(sc.errors...); expected != nil {
			for _, expectedErr := range sc.errors {
				assert.ErrorIs(t, err, expectedErr)
			}
		}

	default:
		assert.NoError(t, err)
		assert.Equal(t, sc.then, trackingStore.Recorded())
	}
}

package scenario

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/command"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
)

// CommandHandlerInit is the entrypoint of the Command Handler scenario API.
//
// A Command Handler scenario can either set the current evaluation context
// by using Given(), or test a "clean-slate" scenario by using When() directly.
type CommandHandlerInit struct{}

// CommandHandler is a scenario type to test the result of Commands
// being handled by a Command Handler.
//
// Command Handlers in Event-sourced systems produce side effects by means
// of Domain Events. This scenario API helps you with testing the Domain Events
// produced by a Command Handler when handling a specific Command.
func CommandHandler() CommandHandlerInit { return CommandHandlerInit{} }

// Given sets the Command Handler scenario preconditions.
//
// Domain Events are used in Event-sourced systems to represent a side effect
// that has taken place in the system. In order to set a given state for the
// system to be in while testing a specific Command evaluation, you should
// specify the Domain Events that have happened thus far.
//
// When you're testing Commands with a clean-slate system, you should either specify
// no Domain Events, or skip directly to When().
func (sc CommandHandlerInit) Given(events ...eventstore.Event) CommandHandlerGiven {
	return CommandHandlerGiven{given: events}
}

// When provides the Command to evaluate.
func (sc CommandHandlerInit) When(cmd eventually.Command) CommandHandlerWhen {
	return CommandHandlerWhen{when: cmd}
}

// CommandHandlerGiven is the state of the scenario once
// a set of Domain Events have been provided using Given(), to represent
// the state of the system at the time of evaluating a Command.
type CommandHandlerGiven struct {
	given []eventstore.Event
}

// When provides the Command to evaluate.
func (sc CommandHandlerGiven) When(cmd eventually.Command) CommandHandlerWhen {
	return CommandHandlerWhen{
		CommandHandlerGiven: sc,
		when:                cmd,
	}
}

// CommandHandlerWhen is the state of the scenario once the state of the
// system and the Command to evaluate have been provided.
type CommandHandlerWhen struct {
	CommandHandlerGiven

	when eventually.Command
}

// Then sets a positive expectation on the scenario outcome, to produce
// the Domain Events provided in input.
//
// The list of Domain Events specified should be ordered as the expected
// order of recording by the Command Handler.
func (sc CommandHandlerWhen) Then(events ...eventstore.Event) CommandHandlerThen {
	return CommandHandlerThen{
		CommandHandlerWhen: sc,
		then:               events,
	}
}

// ThenError sets a negative expectation on the scenario outcome,
// to produce an error value that is similar to the one provided in input.
//
// Error assertion happens using errors.Is(), so the error returned
// by the Command Handler is unwrapped until the cause error to match
// the provided expectation.
func (sc CommandHandlerWhen) ThenError(err error) CommandHandlerThen {
	return CommandHandlerThen{
		CommandHandlerWhen: sc,
		wantError:          true,
		thenError:          err,
	}
}

// ThenFails sets a negative expectation on the scenario outcome,
// to fail the Command execution with no particular assertion on the error returned.
//
// This is useful when the error returned is not important for the Command
// you're trying to test.
func (sc CommandHandlerWhen) ThenFails() CommandHandlerThen {
	return CommandHandlerThen{
		CommandHandlerWhen: sc,
		wantError:          true,
	}
}

// CommandHandlerThen is the state of the scenario once the preconditions
// and expectations have been fully specified.
type CommandHandlerThen struct {
	CommandHandlerWhen

	then      []eventstore.Event
	thenError error
	wantError bool
}

// Using performs the specified expectations of the scenario, using the Command Handler
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
func (sc CommandHandlerThen) Using( //nolint:gocritic
	t *testing.T,
	aggregateType aggregate.Type,
	handlerFactory func(*aggregate.Repository) command.Handler,
) {
	ctx := context.Background()
	store := inmemory.NewEventStore()

	for _, event := range sc.given {
		_, err := store.Append(ctx, event.Stream, eventstore.VersionCheckAny, event.Event)
		if !assert.NoError(t, err) {
			return
		}
	}

	trackingStore := inmemory.NewTrackingEventStore(store)
	repository := aggregate.NewRepository(aggregateType, eventstore.Fused{
		Appender: trackingStore,
		Streamer: store,
	})

	handler := handlerFactory(repository)
	err := handler.Handle(context.Background(), sc.when)

	if !sc.wantError {
		assert.NoError(t, err)
		assert.Equal(t, sc.then, trackingStore.Recorded())

		return
	}

	if !assert.Error(t, err) {
		return
	}

	if sc.thenError != nil {
		assert.ErrorIs(t, err, sc.thenError)
	}
}

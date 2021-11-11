package scenario

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/command"
	"github.com/get-eventually/go-eventually/event"
)

// ProcessManagerInit is the entrypoint of the Process Manager scenario API.
type ProcessManagerInit struct{}

// ProcessManager is a scenario type to test the Domain Commands issued by a
// Process Manager when handling a certain persisted Domain Event.
//
// Process Managers in Event-sourced systems react to incoming Domain Events,
// persisted in the Event Store, and optionally produce "compensating actions"
// in the form of Domain Commands, in order to implement certain business processes
// (hence the name "Process Manager").
func ProcessManager() ProcessManagerInit {
	return ProcessManagerInit{}
}

// Given sets the Process Manager state before the assertion.
//
// The specified Domain Events will be applied on the Process Manager before
// the Domain Event to test (later specified with When()). Depending on the Process
// Manager implementation, applying these Events could either have no meaningful value,
// or update some internal state or Read Model maintained by the Process Manager.
func (ProcessManagerInit) Given(events ...event.Persisted) ProcessManagerGiven {
	return ProcessManagerGiven{
		given: events,
	}
}

// ProcessManagerGiven is the state of the scenario once a set of Domain Events
// have been provided using Given() to represent the state of the Process Manager
// at the time of processing a new Domain Event.
type ProcessManagerGiven struct {
	given []event.Persisted
}

// When provides the persisted Domain Event the Process Manager should process.
func (sc ProcessManagerGiven) When(evt event.Persisted) *ProcessManagerWhen {
	return &ProcessManagerWhen{
		ProcessManagerGiven: sc,
		when:                evt,
	}
}

// ProcessManagerWhen is the state of the scenario once the state of the Process Manager
// and the Domain Event to process have been set.
//
// This type is used with pointer semantics to save some space in memory.
type ProcessManagerWhen struct {
	ProcessManagerGiven

	when event.Persisted
}

// Then sets a positive expectation on the scenario outcome, which should be
// a list of Commands issued as a result of the Domain Event processed.
func (sc *ProcessManagerWhen) Then(commands ...command.Command) ProcessManagerThen {
	return ProcessManagerThen{
		ProcessManagerWhen: sc,
		then:               commands,
	}
}

// ThenError sets a negative expectation on the scenario outcome,
// to produce an error value that is similar to the one provided in input.
//
// Error assertion happens using errors.Is(), so the error returned
// by the Projection is unwrapped until the cause error to match
// the provided expectation.
func (sc *ProcessManagerWhen) ThenError(err error) ProcessManagerThen {
	return ProcessManagerThen{
		ProcessManagerWhen: sc,
		thenError:          err,
		wantError:          true,
	}
}

// ThenFails sets a negative expectation on the scenario outcome,
// to fail the processing of the Domain Event with no particular assertion on the error returned.
//
// This is useful when the error returned is not important for the Domain Event
// you're trying to test.
func (sc *ProcessManagerWhen) ThenFails() ProcessManagerThen {
	return ProcessManagerThen{
		ProcessManagerWhen: sc,
		wantError:          true,
	}
}

// ProcessManagerThen is the state of the scenario once the preconditions
// and expectations have been fully specified.
type ProcessManagerThen struct {
	*ProcessManagerWhen

	then      []command.Command
	thenError error
	wantError bool
}

// ProcessManagerFactory is the factory function used by the Process Manager
// scenario to build the Process Manager type to test.
type ProcessManagerFactory func(cd command.Dispatcher) event.Processor

// Using performs the specified expectations of the scenario,
// using the Process Manager instance produced by the provided factory function.
func (sc ProcessManagerThen) Using(t *testing.T, processManagerFactory ProcessManagerFactory) {
	ctx := context.Background()
	commandDispatcher := command.NewTrackingDispatcher()
	processManager := processManagerFactory(commandDispatcher)

	for _, event := range sc.given {
		if err := processManager.Process(ctx, event); !assert.NoError(t, err) {
			return
		}
	}

	// Flush dispatcher to have clean list of recorded commands.
	commandDispatcher.FlushCommands()

	err := processManager.Process(ctx, sc.when)

	if !sc.wantError {
		assert.NoError(t, err)
		assert.Equal(t, sc.then, commandDispatcher.RecordedCommands())

		return
	}

	if !assert.Error(t, err) {
		return
	}

	if sc.thenError != nil {
		assert.ErrorIs(t, err, sc.thenError)
	}
}

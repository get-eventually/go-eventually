package aggregate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/version"
)

// ScenarioInit is the entrypoint of the Aggregate Root scenario API.
//
// An Aggregate Root scenario can either set the current evaluation context
// by using Given(), or test a "clean-slate" scenario by using When() directly.
type ScenarioInit[I ID, T Root[I]] struct {
	typ Type[I, T]
}

// Scenario is a scenario type to test the result of methods called
// on an Aggregate Root and their effects.
//
// These methods are meant to produce side-effects in the Aggregate Root state, and thus
// in the overall system, enforcing the aggregate invariants represented by the
// Aggregate Root itself.
func Scenario[I ID, T Root[I]](typ Type[I, T]) ScenarioInit[I, T] {
	return ScenarioInit[I, T]{
		typ: typ,
	}
}

// Given allows to set an Aggregate Root state as precondition to the scenario test,
// by specifying ordered Domain Events.
func (sc ScenarioInit[I, T]) Given(events ...event.Persisted) ScenarioGiven[I, T] {
	return ScenarioGiven[I, T]{
		typ:   sc.typ,
		given: events,
	}
}

// When allows to call for a aggregate method/function that creates a new
// Aggregate Root instance.
//
// This method requires a closure that return said new Aggregate Root instance
// (hence why no input parameter) or an error.
func (sc ScenarioInit[I, T]) When(fn func() (T, error)) ScenarioWhen[I, T] {
	return ScenarioWhen[I, T]{
		typ:   sc.typ,
		given: nil,
		fn:    fn,
	}
}

// ScenarioGiven is the state of the scenario once the Aggregate Root
// preconditions have been set through the Scenario().Given() method.
//
// This state gives access to the When() method to specify the aggregate method
// to test using the desired Aggregate Root.
type ScenarioGiven[I ID, T Root[I]] struct {
	typ   Type[I, T]
	given []event.Persisted
}

// When allows to call the aggregate method method on the Aggregate Root instance
// provided by the previous Scenario().Given() call.
//
// The aggregate method must be called inside the required closure parameter.
func (sc ScenarioGiven[I, T]) When(fn func(T) error) ScenarioWhen[I, T] {
	return ScenarioWhen[I, T]{
		typ:   sc.typ,
		given: sc.given,
		fn: func() (T, error) {
			var zeroValue T

			root := sc.typ.Factory()
			eventStream := event.SliceToStream(sc.given)

			if err := RehydrateFromEvents[I](root, eventStream); err != nil {
				return zeroValue, err
			}

			if err := fn(root); err != nil {
				return zeroValue, err
			}

			return root, nil
		},
	}
}

// ScenarioWhen is the state of the scenario once the aggregate method
// to test has been provided through either Scenario().When() or
// Scenario().Given().When() paths.
//
// This state allows to specify the expected outcome on the scenario using either
// Then(), ThenFails(), ThenError() or ThenErrors() methods.
type ScenarioWhen[I ID, T Root[I]] struct {
	typ   Type[I, T]
	given []event.Persisted
	fn    func() (T, error)
}

// Then specifies a successful outcome of the scenario, allowing to assert the
// expected new Aggregate Root version and Domain Events recorded
// during the aggregate method execution.
func (sc ScenarioWhen[I, T]) Then(v version.Version, events ...event.Envelope) ScenarioThen[I, T] {
	return ScenarioThen[I, T]{
		typ:      sc.typ,
		given:    sc.given,
		fn:       sc.fn,
		version:  v,
		expected: events,
		errors:   nil,
		wantErr:  false,
	}
}

// ThenFails specifies an unsuccessful outcome of the scenario, where the aggregate
// method execution fails with an error.
//
// Use this method when there is no need to assert the error returned by the
// aggregate method is of a specific type or value.
func (sc ScenarioWhen[I, T]) ThenFails() ScenarioThen[I, T] {
	return ScenarioThen[I, T]{
		typ:      sc.typ,
		given:    sc.given,
		fn:       sc.fn,
		version:  0,
		expected: nil,
		errors:   nil,
		wantErr:  true,
	}
}

// ThenError specifies an unsuccessful outcome of the scenario, where the aggregate
// method execution fails with an error.
//
// Use this method when you want to assert that the error retured by the aggregate
// method execution is of a specific type or value.
func (sc ScenarioWhen[I, T]) ThenError(err error) ScenarioThen[I, T] {
	return ScenarioThen[I, T]{
		typ:      sc.typ,
		given:    sc.given,
		fn:       sc.fn,
		version:  0,
		expected: nil,
		errors:   []error{err},
		wantErr:  true,
	}
}

// ThenErrors specifies an unsuccessful outcome of the scenario, where the aggregate method
// execution fails with a specific error that wraps multiple error types (e.g. through `errors.Join`).
//
// Use this method when you want to assert that the error returned by the aggregate method
// matches ALL of the errors specified.
func (sc ScenarioWhen[I, T]) ThenErrors(errs ...error) ScenarioThen[I, T] {
	return ScenarioThen[I, T]{
		typ:      sc.typ,
		given:    sc.given,
		fn:       sc.fn,
		version:  0,
		expected: nil,
		errors:   errs,
		wantErr:  true,
	}
}

// ScenarioThen is the state of the scenario where all parameters have
// been set and it's ready to be executed using a testing.T instance.
//
// Use the AssertOn method to run the test scenario.
type ScenarioThen[I ID, T Root[I]] struct {
	typ      Type[I, T]
	given    []event.Persisted
	fn       func() (T, error)
	version  version.Version
	expected []event.Envelope
	errors   []error
	wantErr  bool
}

// AssertOn runs the test scenario using the specified testing.T instance.
func (sc ScenarioThen[I, T]) AssertOn(t *testing.T) {
	switch root, err := sc.fn(); {
	case sc.wantErr:
		assert.Error(t, err)

		if expected := errors.Join(sc.errors...); expected != nil {
			for _, expectedErr := range sc.errors {
				assert.ErrorIs(t, err, expectedErr)
			}
		}

	default:
		assert.NoError(t, err)

		recordedEvents := root.FlushRecordedEvents()
		assert.Equal(t, sc.expected, recordedEvents)
		assert.Equal(t, sc.version, root.Version())
	}
}

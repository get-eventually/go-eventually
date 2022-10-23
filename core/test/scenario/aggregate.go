package scenario

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"
)

// AggregateRootInit is the entrypoint of the Aggregate Root scenario API.
//
// An Aggregate Root scenario can either set the current evaluation context
// by using Given(), or test a "clean-slate" scenario by using When() directly.
type AggregateRootInit[I aggregate.ID, T aggregate.Root[I]] struct {
	typ aggregate.Type[I, T]
}

// AggregateRoot is a scenario type to test the result of methods called
// on an Aggregate Root and their effects.
//
// These methods are meant to produce side-effects in the Aggregate Root state, and thus
// in the overall system, enforcing the domain invariants represented by the
// Aggregate Root itself.
func AggregateRoot[I aggregate.ID, T aggregate.Root[I]](typ aggregate.Type[I, T]) AggregateRootInit[I, T] {
	return AggregateRootInit[I, T]{
		typ: typ,
	}
}

// Given allows to set an Aggregate Root state as precondition to the scenario test,
// by specifying ordered Domain Events.
func (sc AggregateRootInit[I, T]) Given(events ...event.Persisted) AggregateRootGiven[I, T] {
	return AggregateRootGiven[I, T]{
		typ:   sc.typ,
		given: events,
	}
}

// When allows to call for a domain command method/function that creates a new
// Aggregate Root instance.
//
// This method requires a closure that return said new Aggregate Root instance
// (hence why no input parameter) or an error.
func (sc AggregateRootInit[I, T]) When(fn func() (T, error)) AggregateRootWhen[I, T] {
	return AggregateRootWhen[I, T]{
		typ:   sc.typ,
		given: nil,
		fn:    fn,
	}
}

// AggregateRootGiven is the state of the scenario once the Aggregate Root
// preconditions have been set through the AggregateRoot().Given() method.
//
// This state gives access to the When() method to specify the domain command
// to test using the desired Aggregate Root.
type AggregateRootGiven[I aggregate.ID, T aggregate.Root[I]] struct {
	typ   aggregate.Type[I, T]
	given []event.Persisted
}

// When allows to call the domain command method on the Aggregate Root instance
// provided by the previous AggregateRoot().Given() call.
//
// The domain command must be called inside the required closure parameter.
func (sc AggregateRootGiven[I, T]) When(fn func(T) error) AggregateRootWhen[I, T] {
	return AggregateRootWhen[I, T]{
		typ:   sc.typ,
		given: sc.given,
		fn: func() (T, error) {
			var zeroValue T

			root := sc.typ.Factory()
			eventStream := event.SliceToStream(sc.given)

			if err := aggregate.RehydrateFromEvents[I](root, eventStream); err != nil {
				return zeroValue, err
			}

			if err := fn(root); err != nil {
				return zeroValue, err
			}

			return root, nil
		},
	}
}

// AggregateRootWhen is the state of the scenario once the domain command
// to test has been provided through either AggregateRoot().When() or
// AggregateRoot().Given().When() paths.
//
// This state allows to specify the expected outcome on the scenario using either
// Then(), ThenFails() or ThenError() methods.
type AggregateRootWhen[I aggregate.ID, T aggregate.Root[I]] struct {
	typ   aggregate.Type[I, T]
	given []event.Persisted
	fn    func() (T, error)
}

// Then specifies a successful outcome of the scenario, allowing to assert the
// expected new Aggregate Root version and Domain Events recorded
// during the domain command execution.
func (sc AggregateRootWhen[I, T]) Then(v version.Version, events ...event.Envelope) AggregateRootThen[I, T] {
	return AggregateRootThen[I, T]{
		typ:      sc.typ,
		given:    sc.given,
		fn:       sc.fn,
		version:  v,
		expected: events,
	}
}

// ThenFails specifies an unsuccessful outcome of the scenario, where the domain
// command execution fails with an error.
//
// Use this method when there is no need to assert the error returned by the
// domain command is of a specific type or value.
func (sc AggregateRootWhen[I, T]) ThenFails() AggregateRootThen[I, T] {
	return AggregateRootThen[I, T]{
		typ:       sc.typ,
		given:     sc.given,
		fn:        sc.fn,
		wantError: true,
	}
}

// ThenError specifies an unsuccessful outcome of the scenario, where the domain
// command execution fails with an error.
//
// Use this method when you want to assert that the error retured by the domain
// command execution is of a specific type or value.
func (sc AggregateRootWhen[I, T]) ThenError(err error) AggregateRootThen[I, T] {
	return AggregateRootThen[I, T]{
		typ:           sc.typ,
		given:         sc.given,
		fn:            sc.fn,
		expectedError: err,
		wantError:     true,
	}
}

// AggregateRootThen is the state of the scenario where all parameters have
// been set and it's ready to be executed using a testing.T instance.
//
// Use the AssertOn method to run the test scenario.
type AggregateRootThen[I aggregate.ID, T aggregate.Root[I]] struct {
	typ           aggregate.Type[I, T]
	given         []event.Persisted
	fn            func() (T, error)
	version       version.Version
	expected      []event.Envelope
	expectedError error
	wantError     bool
}

// AssertOn runs the test scenario using the specified testing.T instance.
func (sc AggregateRootThen[I, T]) AssertOn(t *testing.T) {
	root, err := sc.fn()

	if !sc.wantError {
		assert.NoError(t, err)

		recordedEvents := root.FlushRecordedEvents()
		assert.Equal(t, sc.expected, recordedEvents)
		assert.Equal(t, sc.version, root.Version())

		return
	}

	assert.Error(t, err)

	if sc.expectedError != nil {
		assert.ErrorIs(t, err, sc.expectedError)
	}
}

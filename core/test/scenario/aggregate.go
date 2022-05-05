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

func (sc AggregateRootInit[I, T]) Given(events ...event.Persisted) AggregateRootGiven[I, T] {
	return AggregateRootGiven[I, T]{
		typ:   sc.typ,
		given: events,
	}
}

func (sc AggregateRootInit[I, T]) When(fn func() (T, error)) AggregateRootWhen[I, T] {
	return AggregateRootWhen[I, T]{
		typ:   sc.typ,
		given: nil,
		fn:    fn,
	}
}

type AggregateRootGiven[I aggregate.ID, T aggregate.Root[I]] struct {
	typ   aggregate.Type[I, T]
	given []event.Persisted
}

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

type AggregateRootWhen[I aggregate.ID, T aggregate.Root[I]] struct {
	typ   aggregate.Type[I, T]
	given []event.Persisted
	fn    func() (T, error)
}

func (sc AggregateRootWhen[I, T]) Then(v version.Version, events ...event.Envelope) AggregateRootThen[I, T] {
	return AggregateRootThen[I, T]{
		typ:      sc.typ,
		given:    sc.given,
		fn:       sc.fn,
		version:  v,
		expected: events,
	}
}

func (sc AggregateRootWhen[I, T]) ThenFails() AggregateRootThen[I, T] {
	return AggregateRootThen[I, T]{
		typ:       sc.typ,
		given:     sc.given,
		fn:        sc.fn,
		wantError: true,
	}
}

func (sc AggregateRootWhen[I, T]) ThenError(err error) AggregateRootThen[I, T] {
	return AggregateRootThen[I, T]{
		typ:           sc.typ,
		given:         sc.given,
		fn:            sc.fn,
		expectedError: err,
		wantError:     true,
	}
}

type AggregateRootThen[I aggregate.ID, T aggregate.Root[I]] struct {
	typ           aggregate.Type[I, T]
	given         []event.Persisted
	fn            func() (T, error)
	version       version.Version
	expected      []event.Envelope
	expectedError error
	wantError     bool
}

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

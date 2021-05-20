package scenario

import (
	"context"
	"testing"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/projection"
	"github.com/get-eventually/go-eventually/query"

	"github.com/stretchr/testify/assert"
)

// ProjectionInit is the entrypoint of the Projection scenario API.
type ProjectionInit struct{}

// Projection is a scenario type to test the result of a Domain Query
// being handled by a Projection, or Domain Read Model.
//
// Projections in Event-sourced systems are updated by means of Domain Events,
// and sometimes used specifically for building optimized Read Models
// to satisfy rather specific Domain Queries.
func Projection() ProjectionInit { return ProjectionInit{} }

// Given sets the Projection state before the assertion.
//
// Domain Events are used in Event-sourced systems to represent a side effect
// that has taken place in the system. Thus, they're also used by Projections
// as a trigger to perform an update on their internal state.
func (s ProjectionInit) Given(events ...eventstore.Event) ProjectionGiven {
	return ProjectionGiven{given: events}
}

// ProjectionGiven is the state of the scenario once a set of Domain Events
// have been provided using Given(), to represent the state of the Read Model/Projection
// at the time of evaluating a Domain Query.
type ProjectionGiven struct {
	given []eventstore.Event
}

// When provides the Domain Query to evaluate using the Read Model/Projection.
func (s ProjectionGiven) When(q query.Query) ProjectionWhen {
	return ProjectionWhen{ProjectionGiven: s, when: q}
}

// ProjectionWhen is the state of the scenario once the state of the Read Model/Projection
// and the Domain Query to evaluate have been provided.
type ProjectionWhen struct {
	ProjectionGiven
	when query.Query
}

// Then sets a positive expectation on the scenario outcome, to produce the Answer
// provided in input.
func (s ProjectionWhen) Then(answer query.Answer) ProjectionThen {
	return ProjectionThen{
		ProjectionWhen: s,
		then:           answer,
	}
}

// ThenError sets a negative expectation on the scenario outcome,
// to produce an error value that is similar to the one provided in input.
//
// Error assertion happens using errors.Is(), so the error returned
// by the Projection is unwrapped until the cause error to match
// the provided expectation.
func (s ProjectionWhen) ThenError(err error) ProjectionThen {
	return ProjectionThen{
		ProjectionWhen: s,
		thenError:      err,
		wantError:      true,
	}
}

// ThenFails sets a negative expectation on the scenario outcome,
// to fail the Query execution with no particular assertion on the error returned.
//
// This is useful when the error returned is not important for the Domain Query
// you're trying to test.
func (s ProjectionWhen) ThenFails() ProjectionThen {
	return ProjectionThen{
		ProjectionWhen: s,
		wantError:      true,
	}
}

// ProjectionThen is the state of the scenario once the preconditions
// and expectations have been fully specified.
type ProjectionThen struct {
	ProjectionWhen
	then      query.Answer
	thenError error
	wantError bool
}

// Using performs the specified expectations of the scenario, using the Projection
// instance produced by the provided factory function.
func (s ProjectionThen) Using(t *testing.T, projectionFactory func() projection.Projection) {
	ctx := context.Background()
	proj := projectionFactory()

	for _, event := range s.given {
		if err := proj.Apply(ctx, event); !assert.NoError(t, err) {
			return
		}
	}

	answer, err := proj.Handle(ctx, s.when)

	if !s.wantError {
		assert.NoError(t, err)
		assert.Equal(t, s.then, answer)

		return
	}

	if !assert.Error(t, err) {
		return
	}

	if s.thenError != nil {
		assert.ErrorIs(t, err, s.thenError)
	}
}

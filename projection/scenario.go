package projection

import (
	"context"
	"errors"
	"testing"

	"github.com/eventually-rs/eventually-go/eventstore"

	"github.com/stretchr/testify/assert"
)

func Given(events ...eventstore.Event) Scenario { return Scenario{given: events} }

type Scenario struct {
	given []eventstore.Event
}

func (s Scenario) When(query Query) ScenarioWhen {
	return ScenarioWhen{given: s.given, when: query}
}

type ScenarioWhen struct {
	given []eventstore.Event
	when  Query
}

func (s ScenarioWhen) Then(answer Answer) ScenarioThen {
	return ScenarioThen{
		given: s.given,
		when:  s.when,
		then:  answer,
	}
}

func (s ScenarioWhen) ThenError(err error) ScenarioThen {
	return ScenarioThen{
		given:     s.given,
		when:      s.when,
		thenError: err,
		wantError: true,
	}
}

func (s ScenarioWhen) ThenFails() ScenarioThen {
	return ScenarioThen{
		given:     s.given,
		when:      s.when,
		wantError: true,
	}
}

type ScenarioThen struct {
	given     []eventstore.Event
	when      Query
	then      Answer
	thenError error
	wantError bool
}

func (s ScenarioThen) Using(t *testing.T, projectionFactory func() Projection) {
	ctx := context.Background()
	projection := projectionFactory()

	for _, event := range s.given {
		if err := projection.Apply(ctx, event); !assert.NoError(t, err) {
			return
		}
	}

	answer, err := projection.Evaluate(ctx, s.when)

	if !s.wantError {
		assert.NoError(t, err)
		assert.Equal(t, s.then, answer)
		return
	}

	if !assert.Error(t, err) {
		return
	}

	if s.thenError != nil && !assert.True(t, errors.Is(err, s.thenError)) {
		t.Log("Unexpected error received:", err)
		return
	}
}

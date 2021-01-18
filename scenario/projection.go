package scenario

import (
	"context"
	"errors"
	"testing"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/projection"
	"github.com/eventually-rs/eventually-go/query"

	"github.com/stretchr/testify/assert"
)

type ProjectionInit struct{}

func Projection() ProjectionInit { return ProjectionInit{} }

func (s ProjectionInit) Given(events ...eventstore.Event) ProjectionGiven {
	return ProjectionGiven{given: events}
}

type ProjectionGiven struct {
	given []eventstore.Event
}

func (s ProjectionGiven) When(query query.Query) ProjectionWhen {
	return ProjectionWhen{ProjectionGiven: s, when: query}
}

type ProjectionWhen struct {
	ProjectionGiven
	when query.Query
}

func (s ProjectionWhen) Then(answer query.Answer) ProjectionThen {
	return ProjectionThen{
		ProjectionWhen: s,
		then:           answer,
	}
}

func (s ProjectionWhen) ThenError(err error) ProjectionThen {
	return ProjectionThen{
		ProjectionWhen: s,
		thenError:      err,
		wantError:      true,
	}
}

func (s ProjectionWhen) ThenFails() ProjectionThen {
	return ProjectionThen{
		ProjectionWhen: s,
		wantError:      true,
	}
}

type ProjectionThen struct {
	ProjectionWhen
	then      query.Answer
	thenError error
	wantError bool
}

func (s ProjectionThen) Using(t *testing.T, projectionFactory func() projection.Projection) {
	ctx := context.Background()
	projection := projectionFactory()

	for _, event := range s.given {
		if err := projection.Apply(ctx, event); !assert.NoError(t, err) {
			return
		}
	}

	answer, err := projection.Handle(ctx, s.when)

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

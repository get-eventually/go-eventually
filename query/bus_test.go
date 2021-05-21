package query_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/query"
)

type domainQuery struct {
	name  string
	value int64
}

type queryHandler struct{}

func (queryHandler) QueryType() query.Query { return domainQuery{} }

func (queryHandler) Handle(ctx context.Context, q query.Query) (query.Answer, error) {
	return q.(domainQuery).value + 1, nil
}

func TestSimpleBus(t *testing.T) {
	t.Run("bus fails if query has not been registered", func(t *testing.T) {
		ctx := context.Background()
		queryBus := query.NewSimpleBus()

		answer, err := queryBus.Dispatch(ctx, domainQuery{
			name:  "fail-query",
			value: 1,
		})

		assert.Nil(t, answer)
		assert.Error(t, err)
	})

	t.Run("bus returns answer if domain query is registered", func(t *testing.T) {
		ctx := context.Background()
		queryBus := query.NewSimpleBus()

		queryBus.Register(queryHandler{})

		answer, err := queryBus.Dispatch(ctx, domainQuery{
			name:  "fail-query",
			value: 1,
		})

		assert.Equal(t, int64(2), answer)
		assert.NoError(t, err)
	})
}

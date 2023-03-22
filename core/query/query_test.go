package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/core/query"
)

var (
	_ query.Query = queryTest1{}
	_ query.Query = queryTest2{}
)

type queryTest1 struct{}

func (queryTest1) Name() string { return "query_test_1" }

type queryTest2 struct{}

func (queryTest2) Name() string { return "query_test_2" }

func TestGenericEnvelope(t *testing.T) {
	query1 := query.ToEnvelope(queryTest1{})
	genericQuery1 := query1.ToGenericEnvelope()

	v1, ok := query.FromGenericEnvelope[queryTest1](genericQuery1)
	assert.Equal(t, query1, v1)
	assert.True(t, ok)

	v2, ok := query.FromGenericEnvelope[queryTest2](genericQuery1)
	assert.Zero(t, v2)
	assert.False(t, ok)
}

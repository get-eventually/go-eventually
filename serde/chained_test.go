package serde_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/serde"
)

func TestChained(t *testing.T) {
	mySerde := serde.Chain(
		myDataSerde,
		serde.NewJSON(func() *myJSONData { return new(myJSONData) }),
	)

	data := myData{
		Enum:      enumFirst,
		Something: 1,
		Else:      "Else",
	}

	expected := []byte(`{"enum":"FIRST","something":1,"else":"Else"}`)

	bytes, err := mySerde.Serialize(data)
	assert.NoError(t, err)
	assert.Equal(t, expected, bytes)

	deserialized, err := mySerde.Deserialize(bytes)
	assert.NoError(t, err)
	assert.Equal(t, data, deserialized)
}

package serdes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/serdes"
)

func TestChained(t *testing.T) {
	serde := serdes.Chain[myData, *myJSONData, []byte](
		myDataSerde,
		serdes.NewJSON(func() *myJSONData { return new(myJSONData) }),
	)

	data := myData{
		Enum:      enumFirst,
		Something: 1,
		Else:      "Else",
	}

	expected := []byte(`{"enum":"FIRST","something":1,"else":"Else"}`)

	bytes, err := serde.Serialize(data)
	assert.NoError(t, err)
	assert.Equal(t, expected, bytes)

	deserialized, err := serde.Deserialize(bytes)
	assert.NoError(t, err)
	assert.Equal(t, data, deserialized)
}

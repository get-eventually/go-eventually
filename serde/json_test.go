package serde_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/serde"
)

type myEnum uint8

const (
	enumFirst myEnum = iota + 1
	enumSecond
	enumThird
)

const (
	enumFirstString  = "FIRST"
	enumSecondString = "SECOND"
	enumThirdString  = "THIRD"
)

type myData struct {
	Enum      myEnum
	Something int64
	Else      string
}

type myJSONData struct {
	Enum      string `json:"enum"`
	Something int64  `json:"something"`
	Else      string `json:"else"`
}

func serializeMyData(data myData) (*myJSONData, error) {
	jsonData := new(myJSONData)

	switch data.Enum {
	case enumFirst:
		jsonData.Enum = enumFirstString
	case enumSecond:
		jsonData.Enum = enumSecondString
	case enumThird:
		jsonData.Enum = enumThirdString
	default:
		return nil, fmt.Errorf("failed to serialize data, unexpected data value, %v", data.Enum)
	}

	jsonData.Something = data.Something
	jsonData.Else = data.Else

	return jsonData, nil
}

func deserializeMyData(jsonData *myJSONData) (myData, error) {
	var data myData

	switch jsonData.Enum {
	case enumFirstString:
		data.Enum = enumFirst
	case enumSecondString:
		data.Enum = enumSecond
	case enumThirdString:
		data.Enum = enumThird
	default:
		return myData{}, fmt.Errorf("failed to deserialize data, unexpected enum value, %v", jsonData.Enum)
	}

	data.Something = jsonData.Something
	data.Else = jsonData.Else

	return data, nil
}

var myDataSerde = serde.Fuse[myData, *myJSONData](
	serde.AsSerializerFunc(serializeMyData),
	serde.AsDeserializerFunc(deserializeMyData),
)

func TestJSON(t *testing.T) {
	myJSONSerde := serde.NewJSON(func() *myJSONData { return new(myJSONData) })

	t.Run("it works with valid data", func(t *testing.T) {
		myJSON := &myJSONData{
			Enum:      "FIRST",
			Something: 1,
			Else:      "Else",
		}

		bytes, err := json.Marshal(myJSON)
		require.NoError(t, err)

		serialized, err := myJSONSerde.Serialize(myJSON)
		assert.NoError(t, err)
		assert.Equal(t, bytes, serialized)

		deserialized, err := myJSONSerde.Deserialize(serialized)
		assert.NoError(t, err)
		assert.Equal(t, myJSON, deserialized)
	})

	t.Run("it fails deserialization of invalid json data", func(t *testing.T) {
		deserialized, err := myJSONSerde.Deserialize([]byte("{"))
		assert.Error(t, err)
		assert.Zero(t, deserialized)
	})

	t.Run("it works also with by-value semantics", func(t *testing.T) {
		type byValue struct {
			Test bool
		}

		mySerde := serde.NewJSON(func() byValue { return byValue{} }) //nolint:exhaustruct // Unnecessary.
		myValue := byValue{Test: true}

		serialized, err := mySerde.Serialize(myValue)
		assert.NoError(t, err)
		assert.NotEmpty(t, serialized)

		deserialized, err := mySerde.Deserialize(serialized)
		assert.NoError(t, err)
		assert.Equal(t, myValue, deserialized)
	})
}

package serdes_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/serdes"
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
	json := new(myJSONData)

	switch data.Enum {
	case enumFirst:
		json.Enum = enumFirstString
	case enumSecond:
		json.Enum = enumSecondString
	case enumThird:
		json.Enum = enumThirdString
	default:
		return nil, fmt.Errorf("failed to serialize data, unexpected data value, %v", data.Enum)
	}

	json.Something = data.Something
	json.Else = data.Else

	return json, nil
}

func deserializeMyData(json *myJSONData) (myData, error) {
	var data myData

	switch json.Enum {
	case enumFirstString:
		data.Enum = enumFirst
	case enumSecondString:
		data.Enum = enumSecond
	case enumThirdString:
		data.Enum = enumThird
	default:
		return myData{}, fmt.Errorf("failed to deserialize data, unexpected enum value, %v", json.Enum)
	}

	data.Something = json.Something
	data.Else = json.Else

	return data, nil
}

var myDataSerde = serde.Fused[myData, *myJSONData]{
	Serializer:   serde.SerializerFunc[myData, *myJSONData](serializeMyData),
	Deserializer: serde.DeserializerFunc[myData, *myJSONData](deserializeMyData),
}

func TestJSON(t *testing.T) {
	myJSONSerde := serdes.NewJSON(func() *myJSONData { return &myJSONData{} })

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

		mySerde := serdes.NewJSON(func() byValue { return byValue{} })
		myValue := byValue{Test: true}

		serialized, err := mySerde.Serialize(myValue)
		assert.NoError(t, err)
		assert.NotEmpty(t, serialized)

		deserialized, err := mySerde.Deserialize(serialized)
		assert.NoError(t, err)
		assert.Equal(t, myValue, deserialized)
	})
}

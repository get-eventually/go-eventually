package serdes

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/get-eventually/go-eventually/core/serde"
)

// NewProtoJSONSerializer returns a serializer function where the input data (T)
// gets serialized to Protobuf JSON byte-array data.
func NewProtoJSONSerializer[T proto.Message]() serde.SerializerFunc[T, []byte] {
	return func(t T) ([]byte, error) {
		data, err := protojson.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("serdes.ProtoJSON: failed to serialize data, %w", err)
		}

		return data, nil
	}
}

// NewProtoJSONDeserializer returns a deserializer function where a byte-array
// is deserialized into a destination model type (T) using Protobuf JSON.
//
// A data factory function is required for creating new instances of type `T`
// (especially if pointer semantics is used).
func NewProtoJSONDeserializer[T proto.Message](factory func() T) serde.DeserializerFunc[T, []byte] {
	return func(data []byte) (T, error) {
		var zeroValue T

		model := factory()

		if err := protojson.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serdes.ProtoJSON: failed to deserialize data, %w", err)
		}

		return model, nil
	}
}

// NewProtoJSON returns a new serde instance where some data (`T`) gets serialized to
// and deserialized from Protobuf JSON.
func NewProtoJSON[T proto.Message](factory func() T) serde.Fused[T, []byte] {
	return serde.Fused[T, []byte]{
		Serializer:   NewProtoJSONSerializer[T](),
		Deserializer: NewProtoJSONDeserializer(factory),
	}
}

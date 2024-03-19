package serde

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// NewProtoJSONSerializer returns a serializer function where the input data (T)
// gets serialized to Protobuf JSON byte-array data.
func NewProtoJSONSerializer[T proto.Message]() SerializerFunc[T, []byte] {
	return func(t T) ([]byte, error) {
		data, err := protojson.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("serde.ProtoJSON: failed to serialize data, %w", err)
		}

		return data, nil
	}
}

// NewProtoJSONDeserializer returns a deserializer function where a byte-array
// is deserialized into a destination model type (T) using Protobuf JSON.
//
// A data factory function is required for creating new instances of type `T`
// (especially if pointer semantics is used).
func NewProtoJSONDeserializer[T proto.Message](factory func() T) DeserializerFunc[T, []byte] {
	return func(data []byte) (T, error) {
		var zeroValue T

		model := factory()

		if err := protojson.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serde.ProtoJSON: failed to deserialize data, %w", err)
		}

		return model, nil
	}
}

// NewProtoJSON returns a new serde instance where some data (`T`) gets serialized to
// and deserialized from Protobuf JSON.
func NewProtoJSON[T proto.Message](factory func() T) Fused[T, []byte] {
	return Fuse[T, []byte](
		NewProtoJSONSerializer[T](),
		NewProtoJSONDeserializer(factory),
	)
}

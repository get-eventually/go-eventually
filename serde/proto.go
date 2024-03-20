package serde

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// NewProtoSerializer returns a serializer function where the input data (T)
// gets serialized to Protobuf byte-array.
func NewProtoSerializer[T proto.Message]() SerializerFunc[T, []byte] {
	return func(t T) ([]byte, error) {
		data, err := proto.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("serde.Proto: failed to serialize data, %w", err)
		}

		return data, nil
	}
}

// NewProtoDeserializer returns a deserializer function where a byte-array
// is deserialized into a destination data type (T) using Protobuf.
//
// A data factory function is required for creating new instances of type `T`
// (especially if pointer semantics is used).
func NewProtoDeserializer[T proto.Message](factory func() T) DeserializerFunc[T, []byte] {
	return func(data []byte) (T, error) {
		var zeroValue T

		model := factory()

		if err := proto.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serde.Proto: failed to deseruialize data, %w", err)
		}

		return model, nil
	}
}

// NewProto returns a new serde instance where some data (`T`) gets serialized to
// and deserialized from a Protobuf byte-array.
func NewProto[T proto.Message](factory func() T) Fused[T, []byte] {
	return Fuse(
		NewProtoSerializer[T](),
		NewProtoDeserializer(factory),
	)
}

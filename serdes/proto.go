package serdes

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/get-eventually/go-eventually/core/serde"
)

// NewProtoSerializer returns a serializer function where the input data (T)
// gets serialized to Protobuf byte-array.
func NewProtoSerializer[T proto.Message]() serde.SerializerFunc[T, []byte] {
	return func(t T) ([]byte, error) {
		data, err := proto.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("serdes.Proto: failed to serialize data, %w", err)
		}

		return data, nil
	}
}

// NewProtoDeserializer returns a deserializer function where a byte-array
// is deserialized into a destination data type (T) using Protobuf.
//
// A data factory function is required for creating new instances of type `T`
// (especially if pointer semantics is used).
func NewProtoDeserializer[T proto.Message](factory func() T) serde.DeserializerFunc[T, []byte] {
	return func(data []byte) (T, error) {
		var zeroValue T

		model := factory()

		if err := proto.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serdes.Proto: failed to deseruialize data, %w", err)
		}

		return model, nil
	}
}

// NewProto returns a new serde instance where some data (`T`) gets serialized to
// and deserialized from a Protobuf byte-array.
func NewProto[T proto.Message](factory func() T) serde.Fused[T, []byte] {
	return serde.Fused[T, []byte]{
		Serializer:   NewProtoSerializer[T](),
		Deserializer: NewProtoDeserializer(factory),
	}
}

package serde

import (
	"encoding/json"
	"fmt"
)

// NewJSONSerializer returns a serializer function where the input data (Src)
// gets serialized to JSON byte-array data.
func NewJSONSerializer[T any]() SerializerFunc[T, []byte] {
	return func(t T) ([]byte, error) {
		data, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("serde.JSON: failed to serialize data, %w", err)
		}

		return data, nil
	}
}

// NewJSONDeserializer returns a deserializer function where a byte-array
// is deserialized into the specified data type.
//
// A data factory function is required for creating new instances of the type
// (especially if pointer semantics is used).
func NewJSONDeserializer[T any](factory func() T) DeserializerFunc[T, []byte] {
	return func(data []byte) (T, error) {
		var zeroValue T

		model := factory()
		if err := json.Unmarshal(data, &model); err != nil {
			return zeroValue, fmt.Errorf("serde.JSON: failed to deserialize data, %w", err)
		}

		return model, nil
	}
}

// NewJSON returns a new serde instance where some data (`T`) gets serialized to
// and deserialized from JSON as byte-array.
func NewJSON[T any](factory func() T) Fused[T, []byte] {
	return Fuse(
		NewJSONSerializer[T](),
		NewJSONDeserializer(factory),
	)
}

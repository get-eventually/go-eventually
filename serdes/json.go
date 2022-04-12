package serdes

import (
	"encoding/json"
	"fmt"

	"github.com/get-eventually/go-eventually/core/serde"
)

// NewJSONSerializer returns a serializer function where the input data (Src)
// gets serialized to JSON byte-array data using a destination model type (Dst).
func NewJSONSerializer[Src any, Dst any](
	serializer serde.Serializer[Src, Dst],
) serde.SerializerFunc[Src, []byte] {
	return func(src Src) ([]byte, error) {
		model, err := serializer.Serialize(src)
		if err != nil {
			return nil, fmt.Errorf("serdes.JSON: failed to serialize through serializer, %w", err)
		}

		data, err := json.Marshal(model)
		if err != nil {
			return nil, fmt.Errorf("serdes.JSON: failed to marshal serializer model using protojson, %w", err)
		}

		return data, nil
	}
}

// NewJSONDeserializer returns a deserializer function where a byte-array
// is deserialized into a destination model type (Dst) using JSON and then converted
// into the desired inpud data structure (Src).
//
// A data factory function is required for creating new instances of type `Dst`
// (especially if pointer semantics is used).
func NewJSONDeserializer[Src any, Dst any](
	deserializer serde.Deserializer[Src, Dst],
	jsonFactory func() Dst,
) serde.DeserializerFunc[Src, []byte] {
	return func(data []byte) (Src, error) {
		var zeroValue Src

		model := jsonFactory()

		if err := json.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serdes.JSON: failed to marshal deserializer model using protojson, %w", err)
		}

		root, err := deserializer.Deserialize(model)
		if err != nil {
			return zeroValue, fmt.Errorf("serdes.JSON: failed to deserialize through deserializer, %w", err)
		}

		return root, nil
	}
}

// NewJSON returns a new serde instance where some data (`Src`) gets serialized to
// and deserialized from JSON using a supporting data structure (`Dst`).
func NewJSON[Src any, Dst any](
	serdes serde.Serde[Src, Dst],
	jsonFactory func() Dst,
) serde.Fused[Src, []byte] {
	return serde.Fused[Src, []byte]{
		Serializer:   NewJSONSerializer[Src, Dst](serdes),
		Deserializer: NewJSONDeserializer[Src, Dst](serdes, jsonFactory),
	}
}

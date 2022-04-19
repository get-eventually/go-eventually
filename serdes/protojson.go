package serdes

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/get-eventually/go-eventually/core/serde"
)

// NewProtoJSONSerializer returns a serializer function where the input data (Src)
// gets serialized to Protobuf JSON byte-array data using a destination model type (Dst).
func NewProtoJSONSerializer[Src any, Dst proto.Message](
	serializer serde.Serializer[Src, Dst],
) serde.SerializerFunc[Src, []byte] {
	return func(src Src) ([]byte, error) {
		model, err := serializer.Serialize(src)
		if err != nil {
			return nil, fmt.Errorf("serdes.ProtoJSON: failed to serialize through serializer, %w", err)
		}

		data, err := protojson.Marshal(model)
		if err != nil {
			return nil, fmt.Errorf("serdes.ProtoJSON: failed to marshal serializer model using protojson, %w", err)
		}

		return data, nil
	}
}

// NewProtoJSONDeserializer returns a deserializer function where a byte-array
// is deserialized into a destination model type (Dst) using Protobuf JSON and then converted
// into the desired inpud data structure (Src).
//
// A data factory function is required for creating new instances of type `Dst`
// (especially if pointer semantics is used).
func NewProtoJSONDeserializer[Src any, Dst proto.Message](
	deserializer serde.Deserializer[Src, Dst],
	protoFactory func() Dst,
) serde.DeserializerFunc[Src, []byte] {
	return func(data []byte) (Src, error) {
		var zeroValue Src

		model := protoFactory()
		if err := protojson.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serdes.ProtoJSON: failed to marshal deserializer model using protojson, %w", err)
		}

		root, err := deserializer.Deserialize(model)
		if err != nil {
			return zeroValue, fmt.Errorf("serdes.ProtoJSON: failed to deserialize through deserializer, %w", err)
		}

		return root, nil
	}
}

// NewProtoJSON returns a new serde instance where some data (`Src`) gets serialized to
// and deserialized from Protobuf JSON using a supporting data structure (`Dst`).
func NewProtoJSON[Src any, Dst proto.Message](
	serdes serde.Serde[Src, Dst],
	protoFactory func() Dst,
) serde.Fused[Src, []byte] {
	return serde.Fused[Src, []byte]{
		Serializer:   NewProtoJSONSerializer[Src, Dst](serdes),
		Deserializer: NewProtoJSONDeserializer[Src, Dst](serdes, protoFactory),
	}
}

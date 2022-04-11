package serdes

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/get-eventually/go-eventually/core/serde"
)

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

func NewProtoJSON[Src any, Dst proto.Message](
	serdes serde.Serde[Src, Dst],
	protoFactory func() Dst,
) serde.Fused[Src, []byte] {
	return serde.Fused[Src, []byte]{
		Serializer:   NewProtoJSONSerializer[Src, Dst](serdes),
		Deserializer: NewProtoJSONDeserializer[Src, Dst](serdes, protoFactory),
	}
}

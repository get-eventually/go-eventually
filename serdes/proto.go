package serdes

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/get-eventually/go-eventually/core/serde"
)

func NewProtoSerializer[Src any, Dst proto.Message](
	serializer serde.Serializer[Src, Dst],
) serde.SerializerFunc[Src, []byte] {
	return func(src Src) ([]byte, error) {
		model, err := serializer.Serialize(src)
		if err != nil {
			return nil, fmt.Errorf("serdes.Proto: failed to serialize through serializer, %w", err)
		}

		data, err := proto.Marshal(model)
		if err != nil {
			return nil, fmt.Errorf("serdes.Proto: failed to marshal serializer model using protojson, %w", err)
		}

		return data, nil
	}
}

func NewProtoDeserializer[Src any, Dst proto.Message](
	deserializer serde.Deserializer[Src, Dst],
	protoFactory func() Dst,
) serde.DeserializerFunc[Src, []byte] {
	return func(data []byte) (Src, error) {
		var zeroValue Src

		model := protoFactory()
		if err := proto.Unmarshal(data, model); err != nil {
			return zeroValue, fmt.Errorf("serdes.Proto: failed to marshal deserializer model using protojson, %w", err)
		}

		root, err := deserializer.Deserialize(model)
		if err != nil {
			return zeroValue, fmt.Errorf("serdes.Proto: failed to deserialize through deserializer, %w", err)
		}

		return root, nil
	}
}

func NewProto[Src any, Dst proto.Message](
	serdes serde.Serde[Src, Dst],
	protoFactory func() Dst,
) serde.Fused[Src, []byte] {
	return serde.Fused[Src, []byte]{
		Serializer:   NewProtoSerializer[Src, Dst](serdes),
		Deserializer: NewProtoDeserializer[Src, Dst](serdes, protoFactory),
	}
}

package message

type Serializer[Src Message, Dst any] interface {
	Serialize(src Src) (Dst, error)
}

type Deserializer[Src any, Dst Message] interface {
	Deserialize(src Src) (Dst, error)
}

type SerializerFunc[Src Message, Dst any] func(src Src) (Dst, error)

func (fn SerializerFunc[Src, Dst]) Serialize(src Src) (Dst, error) {
	return fn(src)
}

type DeserializerFunc[Src any, Dst Message] func(src Src) (Dst, error)

func (fn DeserializerFunc[Src, Dst]) Deserialize(src Src) (Dst, error) {
	return fn(src)
}

type Serde[Src Message, Dst any] struct {
	Serializer[Src, Dst]
	Deserializer[Dst, Src]
}

type GenericSerde[Dst any] struct {
	Serializer[Message, Dst]
	Deserializer[Dst, Message]
}

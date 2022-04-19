package serde

type Serializer[Src any, Dst any] interface {
	Serialize(src Src) (Dst, error)
}

type SerializerFunc[Src any, Dst any] func(src Src) (Dst, error)

func (fn SerializerFunc[Src, Dst]) Serialize(src Src) (Dst, error) { return fn(src) }

type Deserializer[Src any, Dst any] interface {
	Deserialize(dst Dst) (Src, error)
}

type DeserializerFunc[Src any, Dst any] func(dst Dst) (Src, error)

func (fn DeserializerFunc[Src, Dst]) Deserialize(dst Dst) (Src, error) { return fn(dst) }

type Serde[Src any, Dst any] interface {
	Serializer[Src, Dst]
	Deserializer[Src, Dst]
}

type Fused[Src any, Dst any] struct {
	Serializer[Src, Dst]
	Deserializer[Src, Dst]
}

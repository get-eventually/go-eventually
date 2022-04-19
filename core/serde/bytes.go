package serde

type BytesSerializer[Src any] interface {
	Serializer[Src, []byte]
}

type BytesDeserializer[Src any] interface {
	Deserializer[Src, []byte]
}

type Bytes[Src any] interface {
	Serde[Src, []byte]
}

package serde

// BytesSerializer is a specialized Serializer to serialize a Source type
// into a byte array.
type BytesSerializer[Src any] interface {
	Serializer[Src, []byte]
}

// BytesDeserializer is a specialized Deserializer to deserialize a Source type
// from a byte array.
type BytesDeserializer[Src any] interface {
	Deserializer[Src, []byte]
}

// Bytes is a Serde implementation used to serialize a Source type to and
// deserialize it from a byte array.
type Bytes[Src any] interface {
	Serde[Src, []byte]
}

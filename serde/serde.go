package serde

// Serializer is used to serialize a Source type into another Destination type.
type Serializer[Src any, Dst any] interface {
	Serialize(src Src) (Dst, error)
}

// SerializerFunc is a functional implementation of the Serializer interface.
type SerializerFunc[Src any, Dst any] func(src Src) (Dst, error)

// Serialize implements the serde.Serializer interface.
func (fn SerializerFunc[Src, Dst]) Serialize(src Src) (Dst, error) { return fn(src) }

// AsSerializerFunc casts the given serialization function into a
// compatible Serializer interface type.
func AsSerializerFunc[Src, Dst any](f func(src Src) (Dst, error)) SerializerFunc[Src, Dst] {
	return SerializerFunc[Src, Dst](f)
}

// AsInfallibleSerializerFunc casts the given infallible serialization function
// into a compatible Serializer interface type.
func AsInfallibleSerializerFunc[Src, Dst any](f func(src Src) Dst) SerializerFunc[Src, Dst] {
	return SerializerFunc[Src, Dst](func(src Src) (Dst, error) {
		return f(src), nil
	})
}

// Deserializer is used to deserialize a Source type from another Destination type.
type Deserializer[Src any, Dst any] interface {
	Deserialize(dst Dst) (Src, error)
}

// DeserializerFunc is a functional implementation of the Deserializer interface.
type DeserializerFunc[Src any, Dst any] func(dst Dst) (Src, error)

// Deserialize implements the serde.Deserializer interface.
func (fn DeserializerFunc[Src, Dst]) Deserialize(dst Dst) (Src, error) { return fn(dst) }

// AsDeserializerFunc casts the given deserialization function into a
// compatible Deserializer interface type.
func AsDeserializerFunc[Src, Dst any](f func(dst Dst) (Src, error)) DeserializerFunc[Src, Dst] {
	return DeserializerFunc[Src, Dst](f)
}

// AsInfallibleDeserializerFunc casts the given infallible deserialization function
// into a compatible Deserializer interface type.
func AsInfallibleDeserializerFunc[Src, Dst any](f func(dst Dst) Src) DeserializerFunc[Src, Dst] {
	return DeserializerFunc[Src, Dst](func(dst Dst) (Src, error) {
		return f(dst), nil
	})
}

// Serde is used to serialize and deserialize from a Source to a Destination type.
type Serde[Src any, Dst any] interface {
	Serializer[Src, Dst]
	Deserializer[Src, Dst]
}

// Fused provides a convenient way to fuse together different implementations
// of a Serializer and Deserializer, and use it as a Serde.
type Fused[Src any, Dst any] struct {
	Serializer[Src, Dst]
	Deserializer[Src, Dst]
}

// Fuse combines two given Serializer and Deserializer with compatible types
// and returns a Serde implementation through serde.Fused.
func Fuse[Src, Dst any](serializer Serializer[Src, Dst], deserializer Deserializer[Src, Dst]) Fused[Src, Dst] {
	return Fused[Src, Dst]{
		Serializer:   serializer,
		Deserializer: deserializer,
	}
}

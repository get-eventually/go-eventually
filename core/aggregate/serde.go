package aggregate

import (
	"fmt"

	"github.com/get-eventually/go-eventually/core/version"
)

type Serializer[I ID, Src Root[I], Dst any] interface {
	Serialize(src Src) (Dst, error)
}

type Deserializer[I ID, Src any, Dst Root[I]] interface {
	Deserialize(src Src) (Dst, error)
}

type SerializerFunc[I ID, Src Root[I], Dst any] func(src Src) (Dst, error)

func (fn SerializerFunc[I, Src, Dst]) Serialize(src Src) (Dst, error) {
	return fn(src)
}

type DeserializerFunc[I ID, Src any, Dst Root[I]] func(src Src) (Dst, error)

func (fn DeserializerFunc[I, Src, Dst]) Deserialize(src Src) (Dst, error) {
	return fn(src)
}

type Serde[I ID, Src Root[I], Dst any] struct {
	Serializer[I, Src, Dst]
	Deserializer[I, Dst, Src]
}

func RehydrateFromState[I ID, Src any, Dst Root[I]](
	v version.Version,
	src Src,
	deserializer Deserializer[I, Src, Dst],
) (Dst, error) {
	dst, err := deserializer.Deserialize(src)
	if err != nil {
		return dst, fmt.Errorf("aggregate.RehydrateFromState: failed to deserialize source into destination root, %w", err)
	}

	dst.setVersion(v)

	return dst, nil
}

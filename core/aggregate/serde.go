package aggregate

import (
	"fmt"

	"github.com/get-eventually/go-eventually/core/version"
)

type Serializer[I ID, Dst Root[I], Src any] interface {
	Serialize(dst Dst) (Src, error)
}

type Deserializer[I ID, Dst Root[I], Src any] interface {
	Deserialize(v version.Version, src Src) (Dst, error)
}

func RehydrateFromState[I ID, Dst Root[I], Src any](
	v version.Version,
	src Src,
	deserializer Deserializer[I, Dst, Src],
) (Dst, error) {
	dst, err := deserializer.Deserialize(v, src)
	if err != nil {
		return dst, fmt.Errorf("aggregate.RehydrateFromState: failed to deserialize source into destination root, %w", err)
	}

	dst.setVersion(v)

	return dst, nil
}

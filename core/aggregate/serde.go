package aggregate

import (
	"fmt"

	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

func RehydrateFromState[I ID, Src Root[I], Dst any](
	v version.Version,
	dst Dst,
	deserializer serde.Deserializer[Src, Dst],
) (Src, error) {
	var zeroValue Src

	src, err := deserializer.Deserialize(dst)
	if err != nil {
		return zeroValue, fmt.Errorf("aggregate.RehydrateFromState: failed to deserialize source into destination root, %w", err)
	}

	src.setVersion(v)

	return src, nil
}

package aggregate

import (
	"fmt"

	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

// RehydrateFromState rehydrates an aggregate.Root instance
// using a state type, typically coming from an external state type (e.g. Protobuf type)
// and aggregate.Repository implementation (e.g. eventuallypostgres.AggregateRepository).
func RehydrateFromState[I ID, Src Root[I], Dst any](
	v version.Version,
	dst Dst,
	deserializer serde.Deserializer[Src, Dst],
) (Src, error) {
	var zeroValue Src

	src, err := deserializer.Deserialize(dst)
	if err != nil {
		return zeroValue, fmt.Errorf("aggregate.RehydrateFromState: failed to deserialize src into dst root, %w", err)
	}

	src.setVersion(v)

	return src, nil
}

package serde

import (
	"fmt"
)

// Chained is a serde type that allows to chain two separate serdes,
// to map from an Src to a Dst type, using a common supporting type in the middle (Mid).
type Chained[Src any, Mid any, Dst any] struct {
	first  Serde[Src, Mid]
	second Serde[Mid, Dst]
}

// Serialize implements the serde.Serializer interface.
func (s Chained[Src, Mid, Dst]) Serialize(src Src) (Dst, error) {
	var zeroValue Dst

	mid, err := s.first.Serialize(src)
	if err != nil {
		return zeroValue, fmt.Errorf("serde.Chained: first stage serializer failed, %w", err)
	}

	dst, err := s.second.Serialize(mid)
	if err != nil {
		return zeroValue, fmt.Errorf("serde.Chained: second stage serializer failed, %w", err)
	}

	return dst, nil
}

// Deserialize implements the serde.Deserializer interface.
func (s Chained[Src, Mid, Dst]) Deserialize(dst Dst) (Src, error) {
	var zeroValue Src

	mid, err := s.second.Deserialize(dst)
	if err != nil {
		return zeroValue, fmt.Errorf("serde.Chained: first stage deserializer failed, %w", err)
	}

	src, err := s.first.Deserialize(mid)
	if err != nil {
		return zeroValue, fmt.Errorf("serde.Chained: second stage deserializer failed, %w", err)
	}

	return src, nil
}

// Chain chains together two serdes to build a new serde instance to map from Src to Dst types.
func Chain[Src any, Mid any, Dst any](first Serde[Src, Mid], second Serde[Mid, Dst]) Chained[Src, Mid, Dst] {
	return Chained[Src, Mid, Dst]{
		first:  first,
		second: second,
	}
}

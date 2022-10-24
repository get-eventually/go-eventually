package serdes

import (
	"fmt"

	"github.com/get-eventually/go-eventually/core/serde"
)

type Chained[Src any, Mid any, Dst any] struct {
	first  serde.Serde[Src, Mid]
	second serde.Serde[Mid, Dst]
}

func (s Chained[Src, Mid, Dst]) Serialize(src Src) (Dst, error) {
	var zeroValue Dst

	mid, err := s.first.Serialize(src)
	if err != nil {
		return zeroValue, fmt.Errorf("serdes.Chained: first stage serializer failed, %w", err)
	}

	dst, err := s.second.Serialize(mid)
	if err != nil {
		return zeroValue, fmt.Errorf("serdes.Chained: second stage serializer failed, %w", err)
	}

	return dst, nil
}

func (s Chained[Src, Mid, Dst]) Deserialize(dst Dst) (Src, error) {
	var zeroValue Src

	mid, err := s.second.Deserialize(dst)
	if err != nil {
		return zeroValue, fmt.Errorf("serdes.Chained: first stage deserializer failed, %w", err)
	}

	src, err := s.first.Deserialize(mid)
	if err != nil {
		return zeroValue, fmt.Errorf("serdes.Chained: second stage deserializer failed, %w", err)
	}

	return src, nil
}

func Chain[Src any, Mid any, Dst any](
	first serde.Serde[Src, Mid],
	second serde.Serde[Mid, Dst],
) Chained[Src, Mid, Dst] {
	return Chained[Src, Mid, Dst]{first: first, second: second}
}

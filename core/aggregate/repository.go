package aggregate

import (
	"context"
	"fmt"
)

var ErrRootNotFound = fmt.Errorf("aggregate: root not found")

type Getter[I ID, T Root[I]] interface {
	Get(ctx context.Context, id I) (T, error)
}

type Saver[I ID, T Root[I]] interface {
	Save(ctx context.Context, root T) error
}

type Repository[I ID, T Root[I]] interface {
	Getter[I, T]
	Saver[I, T]
}

type FusedRepository[I ID, T Root[I]] interface {
	Getter[I, T]
	Saver[I, T]
}

package aggregate

import (
	"context"
	"fmt"
)

// ErrRootNotFound is returned when the Aggregate Root requested
// through a Repository was not found.
var ErrRootNotFound = fmt.Errorf("aggregate: root not found")

// Getter is an Aggregate Repository interface component,
// that can be used for retrieving Aggregate Roots from some storage.
type Getter[I ID, T Root[I]] interface {
	Get(ctx context.Context, id I) (T, error)
}

// Saver is an Aggregate Repository interface component,
// that can be used for storing Aggregate Roots in some storage.
type Saver[I ID, T Root[I]] interface {
	Save(ctx context.Context, root T) error
}

// Repository is an interface used to get Aggregate Roots from and save them to
// some kind of storage, depending on the implementation.
type Repository[I ID, T Root[I]] interface {
	Getter[I, T]
	Saver[I, T]
}

// FusedRepository is a convenience type that can be used to fuse together
// different implementations for the Getter and Saver Repository interface components.
type FusedRepository[I ID, T Root[I]] struct {
	Getter[I, T]
	Saver[I, T]
}

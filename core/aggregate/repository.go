package aggregate

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/core/event"
)

// ErrRootNotFound is returned when the Aggregate Root requested
// through a Repository was not found.
var ErrRootNotFound = fmt.Errorf("aggregate: root not found")

// Getter is an Aggregate Repository interface component,
// that can be used for retrieving Aggregate Roots from some storage.
type Getter[I ID, Evt event.Event, T Root[I, Evt]] interface {
	Get(ctx context.Context, id I) (T, error)
}

// Saver is an Aggregate Repository interface component,
// that can be used for storing Aggregate Roots in some storage.
type Saver[I ID, Evt event.Event, T Root[I, Evt]] interface {
	Save(ctx context.Context, root T) error
}

// Repository is an interface used to get Aggregate Roots from and save them to
// some kind of storage, depending on the implementation.
type Repository[I ID, Evt event.Event, T Root[I, Evt]] interface {
	Getter[I, Evt, T]
	Saver[I, Evt, T]
}

// FusedRepository is a convenience type that can be used to fuse together
// different implementations for the Getter and Saver Repository interface components.
type FusedRepository[I ID, Evt event.Event, T Root[I, Evt]] struct {
	Getter[I, Evt, T]
	Saver[I, Evt, T]
}

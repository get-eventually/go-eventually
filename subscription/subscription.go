package subscription

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
)

type Subscription interface {
	Name() string
	Start(context.Context, eventstore.EventStream) error
}

type Checkpointer interface {
	Get(context.Context, string) (int64, error)
	Store(context.Context, string, int64) error
}

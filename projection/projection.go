package projection

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
)

type Query interface{}

type Answer interface{}

type Projection interface {
	Apply(context.Context, eventstore.Event) error
	Evaluate(context.Context, Query) (Answer, error)
}

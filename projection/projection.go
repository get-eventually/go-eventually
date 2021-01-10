package projection

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/query"
)

type Applier interface {
	Apply(context.Context, eventstore.Event) error
}

type Projection interface {
	query.Handler
	Applier
}

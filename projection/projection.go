package projection

import (
	"context"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/query"
)

// Applier is a segregated Projection interface that focuses on applying
// Events to a Projection state to trigger a side effect.
//
// Applier implementations do not necessarily require to be Domain Read Models,
// and thus implementing the query.Handler interface.
//
// Examples of pure Appliers are Process Managers, which are long running
// Event listeners that trigger side effects by orchestrating compensating actions.
//
// Use a Runner with a tailored Subscription type to make use
// of an Applier instance.
type Applier interface {
	Apply(context.Context, eventstore.Event) error
}

// ApplierFunc is a functional Applier type implementation.
type ApplierFunc func(ctx context.Context, event eventstore.Event) error

// Apply executes the function pointed by the current value.
func (fn ApplierFunc) Apply(ctx context.Context, event eventstore.Event) error { return fn(ctx, event) }

// Projection represents a Domain Read Model, whose state is updated
// through means of persisted Events coming form the Event Store (the Applier interface),
// and can be accessed using a Domain Query (the query.Handler interface).
//
// Use a Runner with a tailored Subscription type to make use of a Projection.
// Usually, if long-lived and stored in persistent storage, it's suggested to use
// a Catch-up Subscription with a Projection, using a persisted Checkpointer
// to survive application restarts.
type Projection interface {
	query.Handler
	Applier
}

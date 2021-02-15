package query

import (
	"context"
	"fmt"
	"reflect"
)

// Dispatcher represents a component that routes Domain Queries into
// their appropriate Query Handlers.
//
// Differently from Command Handlers, Query Handlers return an Answer to
// the Domain Query dispatched, which usually is the Domain Read Model
// for a specific query type.
//
// Implementations can be synchronous or asynchronous, although it is generally
// used as a synchronous component, i.e. the call to Dispatch returns when
// the query has been resolved.
type Dispatcher interface {
	Dispatch(context.Context, Query) (Answer, error)
}

// SimpleBus is a simple Query Bus implementation, synchronous and that
// works in-memory by registering Query Handlers and dispatching
// Domain Queries, the Bus will take care of routing the Query to the
// appropriate Query Handler.
//
// Use NewSimpleBus to create a new SimpleBus instance.
type SimpleBus struct {
	handlers map[reflect.Type]Handler
}

// NewSimpleBus returns a new instance of SimpleBus.
func NewSimpleBus() *SimpleBus {
	return &SimpleBus{handlers: make(map[reflect.Type]Handler)}
}

// Register adds the specified Query Handler to the SimpleBus routing table,
// associated with the Query Handler's Domain Query type, so that calls
// to Dispatch using the specified Domain Query type will be routed
// to this Query Handler.
//
// Please note, when registering multiple Handlers accepting the same Query type,
// the last registration call will overwrite any previous Handler registration,
// as SimpleBus, intentionally, does not support any kind of voting mechanism.
func (b *SimpleBus) Register(handler Handler) {
	typ := reflect.TypeOf(handler.QueryType())
	b.handlers[typ] = handler
}

// Dispatch routes the provided Domain Query to the appropriate Query Handler
// registered in the Bus, if any, and returns the produced Answer.
//
// If the submitted Query has not been registered by any Handler, an
// error will be returned instead.
func (b SimpleBus) Dispatch(ctx context.Context, query Query) (Answer, error) {
	typ := reflect.TypeOf(query)

	handler, ok := b.handlers[typ]
	if !ok {
		return nil, fmt.Errorf("query.Bus: no handler registered for type %s", typ.Name())
	}

	return handler.Handle(ctx, query)
}

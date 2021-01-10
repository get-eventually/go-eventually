package query

import (
	"context"
	"fmt"
	"reflect"
)

type SimpleBus struct {
	handlers map[reflect.Type]Handler
}

func NewSimpleBus() *SimpleBus {
	return &SimpleBus{handlers: make(map[reflect.Type]Handler)}
}

func (b *SimpleBus) Register(handler Handler) {
	typ := reflect.TypeOf(handler.QueryType())
	b.handlers[typ] = handler
}

func (b SimpleBus) Dispatch(ctx context.Context, query Query) (Answer, error) {
	typ := reflect.TypeOf(query)

	handler, ok := b.handlers[typ]
	if !ok {
		return nil, fmt.Errorf("query.Bus: no handler registered for type %s", typ.Name())
	}

	return handler.Handle(ctx, query)
}

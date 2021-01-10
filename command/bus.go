package command

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
	typ := reflect.TypeOf(handler.CommandType())
	b.handlers[typ] = handler
}

func (b SimpleBus) Dispatch(ctx context.Context, cmd Command) error {
	typ := reflect.TypeOf(cmd)

	handler, ok := b.handlers[typ]
	if !ok {
		return fmt.Errorf("command.Bus: no handler registered for type %s", typ.Name())
	}

	return handler.Handle(ctx, cmd)
}

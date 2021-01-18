package command

import (
	"context"
	"fmt"
	"reflect"

	"github.com/eventually-rs/eventually-go"
)

type Dispatcher interface {
	Dispatch(context.Context, eventually.Command) error
}

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

func (b SimpleBus) Dispatch(ctx context.Context, cmd eventually.Command) error {
	typ := reflect.TypeOf(cmd.Payload)

	handler, ok := b.handlers[typ]
	if !ok {
		return fmt.Errorf("command.SimpleBus: no handler registered for type %s", typ.Name())
	}

	return handler.Handle(ctx, cmd)
}

package command

import (
	"context"

	"github.com/eventually-rs/eventually-go"
)

type Command interface{}

type Handler interface {
	CommandType() Command
	Handle(context.Context, eventually.Command) error
}

type HandlerFunc func(context.Context, Command) error

func (fn HandlerFunc) Handle(ctx context.Context, cmd eventually.Command) error {
	return fn(ctx, cmd)
}

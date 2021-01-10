package command

import (
	"context"
)

type Command interface {
	AggregateID() string
}

type Handler interface {
	CommandType() Command
	Handle(context.Context, Command) error
}

type HandlerFunc func(context.Context, Command) error

func (fn HandlerFunc) Handle(ctx context.Context, cmd Command) error { return fn(ctx, cmd) }

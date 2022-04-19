package command

import (
	"context"

	"github.com/get-eventually/go-eventually/core/message"
)

type Command message.Message

type Envelope[T Command] message.Envelope[T]

type Handler[T Command] interface {
	Handle(ctx context.Context, cmd Envelope[T]) error
}

type HandlerFunc[T Command] func(context.Context, Envelope[T]) error

func (fn HandlerFunc[T]) Handle(ctx context.Context, cmd Envelope[T]) error {
	return fn(ctx, cmd)
}

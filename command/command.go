package command

import (
	"context"

	"github.com/get-eventually/go-eventually/message"
)

// Command is a specific kind of Message that represents an intent.
// Commands should be phrased in the present, imperative tense, such as "ActivateUser" or "CreateOrder".
type Command message.Message

// Envelope carries both a Command and some optional Metadata attached to it.
type Envelope[T Command] message.Envelope[T]

// ToGenericEnvelope returns a GenericEnvelope version of the current Envelope instance.
func (cmd Envelope[T]) ToGenericEnvelope() GenericEnvelope {
	return GenericEnvelope{
		Message:  cmd.Message,
		Metadata: cmd.Metadata,
	}
}

// Handler is the interface that defines a Command Handler,
// a component that receives a specific kind of Command
// and executes the business logic related to that particular Command.
type Handler[T Command] interface {
	Handle(ctx context.Context, cmd Envelope[T]) error
}

// HandlerFunc is a functional type that implements the Handler interface.
// Useful for testing and stateless Handlers.
type HandlerFunc[T Command] func(context.Context, Envelope[T]) error

// Handle handles the provided Command through the functional Handler.
func (fn HandlerFunc[T]) Handle(ctx context.Context, cmd Envelope[T]) error {
	return fn(ctx, cmd)
}

// GenericEnvelope is a Command Envelope that depends solely on the Command interface,
// not a specific generic Command type.
type GenericEnvelope Envelope[Command]

// FromGenericEnvelope attempts to type-cast a GenericEnvelope instance into
// a strongly-typed Command Envelope.
//
// A boolean guard is returned to signal whether the type-casting was successful
// or not.
func FromGenericEnvelope[T Command](cmd GenericEnvelope) (Envelope[T], bool) {
	if v, ok := cmd.Message.(T); ok {
		return Envelope[T]{
			Message:  v,
			Metadata: cmd.Metadata,
		}, true
	}

	return Envelope[T]{}, false
}

// ToEnvelope is a convenience function that wraps the provided Command type
// into an Envelope, with no metadata attached to it.
func ToEnvelope[T Command](cmd T) Envelope[T] {
	return Envelope[T]{
		Message:  cmd,
		Metadata: nil,
	}
}

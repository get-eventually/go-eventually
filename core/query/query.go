package query

import (
	"context"

	"github.com/get-eventually/go-eventually/core/message"
)

// Query is a specific kind of Message that represents the a request for information.
type Query message.Message

// Envelope carries both a Query and some optional Metadata attached to it.
type Envelope[T Query] message.Envelope[T]

// ToGenericEnvelope returns a GenericEnvelope version of the current Envelope instance.
func (cmd Envelope[T]) ToGenericEnvelope() GenericEnvelope {
	return GenericEnvelope{
		Message:  cmd.Message,
		Metadata: cmd.Metadata,
	}
}

// Handler is the interface that defines a Query Handler,
// a component that receives a specific kind of Query and executes it to return
// the desired output.
type Handler[T Query, R any] interface {
	Handle(ctx context.Context, query Envelope[T]) (R, error)
}

// HandlerFunc is a functional type that implements the Handler interface.
// Useful for testing and stateless Handlers.
type HandlerFunc[T Query, R any] func(context.Context, Envelope[T]) (R, error)

// Handle handles the provided Query through the functional Handler.
func (fn HandlerFunc[T, R]) Handle(ctx context.Context, cmd Envelope[T]) (R, error) {
	return fn(ctx, cmd)
}

// GenericEnvelope is a Query Envelope that depends solely on the Query interface,
// not a specific generic Query type.
type GenericEnvelope Envelope[Query]

// FromGenericEnvelope attempts to type-cast a GenericEnvelope instance into
// a strongly-typed Query Envelope.
//
// A boolean guard is returned to signal whether the type-casting was successful
// or not.
func FromGenericEnvelope[T Query](cmd GenericEnvelope) (Envelope[T], bool) {
	if v, ok := cmd.Message.(T); ok {
		return Envelope[T]{
			Message:  v,
			Metadata: cmd.Metadata,
		}, true
	}

	return Envelope[T]{}, false
}

// ToEnvelope is a convenience function that wraps the provided Query type
// into an Envelope, with no metadata attached to it.
func ToEnvelope[T Query](cmd T) Envelope[T] {
	return Envelope[T]{
		Message:  cmd,
		Metadata: nil,
	}
}

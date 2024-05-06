// Package query provides support and utilities to handle and implement
// Domain Queries in your application.
package query

import (
	"context"

	"github.com/get-eventually/go-eventually/message"
)

// Query represents a Domain Query, a request for information.
// Queries should be phrased in the present, imperative tense, such as "ListUsers".
type Query message.Message

// Envelope represents a message containing a Domain Query,
// and optionally includes additional fields in the form of Metadata.
type Envelope[T Query] message.Envelope[T]

// Handler is the interface that defines a Query Handler.
//
// Handler accepts a specific kind of Query, evaluates it
// and returns the desired Result.
type Handler[T Query, R any] interface {
	Handle(ctx context.Context, query Envelope[T]) (R, error)
}

// ToEnvelope is a convenience function that wraps the provided Query type
// into an Envelope, with no metadata attached to it.
func ToEnvelope[T Query](query T) Envelope[T] {
	return Envelope[T]{
		Message:  query,
		Metadata: nil,
	}
}

// HandlerFunc is a functional type that implements the Handler interface.
// Useful for testing and stateless Handlers.
type HandlerFunc[T Query, R any] func(ctx context.Context, query Envelope[T]) (R, error)

// Handle implements xquery.Handler.
func (f HandlerFunc[T, R]) Handle(ctx context.Context, query Envelope[T]) (R, error) {
	return f(ctx, query)
}

package query

import "context"

type Query interface{}

type Answer interface{}

type Handler interface {
	QueryType() Query
	Handle(context.Context, Query) (Answer, error)
}

type HandlerFunc func(context.Context, Query) (Answer, error)

func (fn HandlerFunc) Handle(ctx context.Context, q Query) (Answer, error) { return fn(ctx, q) }

package query

import "context"

// Query is a marker interface for Domain Query types.
type Query interface{}

// Answer is a marker interface for answers to Domain Queries, returned
// by a Query Handler.
type Answer interface{}

// Handler is a component capable of handling Domain Queries, resolving
// them into an Answer, typically a Domain Read Model.
//
// An Handler implementation specifies their supported Query type
// using the QueryType method. Please, make sure the type returned by this method
// reflects the same type of the Query submitted in your application.
type Handler interface {
	QueryType() Query
	Handle(context.Context, Query) (Answer, error)
}

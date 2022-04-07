package aggregate

import "fmt"

// ID represents an Aggregate ID type.
//
// Aggregate IDs should be able to be marshaled into a string format,
// in order to be saved onto a named Event Stream.
type ID interface {
	fmt.Stringer
}

// StringID is a string-typed Aggregate ID.
type StringID string

func (id StringID) String() string { return string(id) }

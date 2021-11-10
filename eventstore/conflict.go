package eventstore

import "fmt"

// ConflictError is an error returned by an Event Store when appending
// some events using an expected Event Stream version that does not match
// the current state of the Event Stream.
type ConflictError struct {
	Expected int64
	Actual   int64
}

func (err ConflictError) Error() string {
	return fmt.Sprintf(
		"conflict detected: expected stream version: %d, actual: %d",
		err.Expected,
		err.Actual,
	)
}

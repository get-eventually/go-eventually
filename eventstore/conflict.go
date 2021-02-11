package eventstore

import "fmt"

// ErrConflict is an error returned by an Event Store when appending
// some events using an expected Event Stream version that does not match
// the current state of the Event Stream.
type ErrConflict struct {
	Expected int64
	Actual   int64
}

func (err ErrConflict) Error() string {
	return fmt.Sprintf(
		"conflict detected: expected stream version: %d, actual: %d",
		err.Expected,
		err.Actual,
	)
}

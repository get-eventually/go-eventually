package version

import (
	"fmt"
)

// Any avoids optimistic concurrency checks when requiring a version.Check instance.
var Any = CheckAny{}

// Check can be used to perform optimistic concurrency checks when writing to
// the Event Store using the event.Appender interface.
type Check interface {
	isVersionCheck()
}

// CheckAny is a Check variant that will avoid optimistic concurrency checks when used.
type CheckAny struct{}

func (CheckAny) isVersionCheck() {}

// CheckExact is a Check variant that will ensure the specified version is the current one
// (typically used when needing to check the version of an Event Stream).
type CheckExact Version

func (CheckExact) isVersionCheck() {}

// ErrConflict is an error returned by an Event Store when appending
// some events using an expected Event Stream version that does not match
// the current state of the Event Stream.
type ErrConflict struct {
	Expected Version
	Actual   Version
}

func (err ErrConflict) Error() string {
	return fmt.Sprintf(
		"version.Check: conflict detected; expected stream version: %d, actual: %d",
		err.Expected,
		err.Actual,
	)
}

package stream

// ID represents the unique identifier for an Event Stream.
type ID struct {
	// Type is the type, or category, of the Event Stream to which this
	// Event belong. Usually, this is the name of the Aggregate type.
	Type string

	// Name is the name of the Event Stream to which this Event belong.
	// Usually, this is the string representation of the Aggregate id.
	Name string
}

// Target represents one or more Event Streams using different discriminators.
//
// This is a sealed interface and implementations are only provided by this package.
// If you want to add support for an additional target, add it in this file, implement
// this interface and make sure users of this interface are updated correctly (e.g. Event Stores).
type Target interface {
	isStreamTarget()
}

// All selects all existing Event Streams.
type All struct{}

func (All) isStreamTarget() {}

// ByType selects all Event Streams with the specified Stream Type identifier.
type ByType string

func (ByType) isStreamTarget() {}

// ByTypes selects all Event Streams with the Stream Type identifiers in the provided list.
type ByTypes []string

func (ByTypes) isStreamTarget() {}

// ByID selects a single Event Stream identified by the provided stream.ID.
type ByID ID

func (ByID) isStreamTarget() {}

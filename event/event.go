package event

import (
	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/event/version"
)

// Event is a Message representing some Domain information that has happened
// in the past, which is of vital information to the Domain itself.
//
// Event type names should be phrased in the past tense, to enforce the notion
// of "information happened in the past".
type Event eventually.Message

// StreamID represents the unique identifier for an Event Stream.
type StreamID struct {
	// Type is the type, or category, of the Event Stream to which this
	// Event belong. Usually, this is the name of the Aggregate type.
	Type string

	// Name is the name of the Event Stream to which this Event belong.
	// Usually, this is the string representation of the Aggregate id.
	Name string
}

// Persisted represents an Domain Event that has been persisted into the Event Store.
type Persisted struct {
	Event

	Stream  StreamID
	Version version.Version
}

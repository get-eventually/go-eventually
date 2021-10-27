package event

import (
	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/event/version"
)

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

type Persisted struct {
	Event

	Stream         StreamID
	Version        version.Version
	SequenceNumber version.SequenceNumber
}

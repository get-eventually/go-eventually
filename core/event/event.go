package event

import (
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/version"
)

// Event is a Message representing some Domain information that has happened
// in the past, which is of vital information to the Domain itself.
//
// Event type names should be phrased in the past tense, to enforce the notion
// of "information happened in the past".
type Event message.Message

type Envelope message.GenericEnvelope

type StreamID string

// Persisted represents an Domain Event that has been persisted into the Event Store.
type Persisted struct {
	StreamID
	version.Version
	Envelope
}

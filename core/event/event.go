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

// Envelope contains a Domain Event and possible metadata associated to it.
//
// Due to lack of sum types (a.k.a enum types), Events cannot currently
// take advantage of the new generics feature introduced with Go 1.18.
type Envelope message.GenericEnvelope

// StreamID identifies an Event Stream, which is a log of ordered Domain Events.
type StreamID string

// Persisted represents an Domain Event that has been persisted into the Event Store.
type Persisted struct {
	StreamID
	version.Version
	Envelope
}

func ToEnvelope(event Event) Envelope {
	return Envelope{
		Message:  event,
		Metadata: nil,
	}
}

func ToEnvelopes(events ...Event) []Envelope {
	envelopes := make([]Envelope, 0, len(events))

	for _, event := range events {
		envelopes = append(envelopes, Envelope{
			Message:  event,
			Metadata: nil,
		})
	}

	return envelopes
}

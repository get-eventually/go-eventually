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
type Envelope[T Event] message.Envelope[T]

// StreamID identifies an Event Stream, which is a log of ordered Domain Events.
type StreamID string

// Persisted represents an Domain Event that has been persisted into the Event Store.
type Persisted[T Event] struct {
	StreamID
	version.Version
	Envelope[T]
}

// ToEnvelope returns an Envelope instance with the provided Event
// instance and no Metadata.
func ToEnvelope[T Event](event T) Envelope[T] {
	return Envelope[T]{
		Message:  event,
		Metadata: nil,
	}
}

// ToEnvelopes returns a list of Envelopes from a list of Events.
// The returned Envelopes have no Metadata.
func ToEnvelopes[T Event](events ...T) []Envelope[T] {
	envelopes := make([]Envelope[T], 0, len(events))

	for _, event := range events {
		envelopes = append(envelopes, ToEnvelope(event))
	}

	return envelopes
}

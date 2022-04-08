package message

// Message is a Message payload.
//
// Each payload should have a unique name identifier, that can be used
// to uniquely route a message to its type.
type Message interface {
	Name() string
}

// Metadata contains some data related to a Message that are not functional
// for the Message itself, but instead functioning as supporting information
// to provide additional context.
type Metadata map[string]string

// With returns a new Metadata reference holding the value addressed using
// the specified key.
func (m Metadata) With(key, value string) Metadata {
	if m == nil {
		m = make(Metadata)
	}

	m[key] = value

	return m
}

// Merge merges the other Metadata provided in input with the current map.
// Returns a pointer to the extended metadata map.
func (m Metadata) Merge(other Metadata) Metadata {
	if m == nil {
		return other
	}

	for k, v := range other {
		m[k] = v
	}

	return m
}

type GenericEnvelope Envelope[Message]

type Envelope[T Message] struct {
	Message  T
	Metadata Metadata
}

func (e Envelope[T]) ToGenericEnvelope() GenericEnvelope {
	return GenericEnvelope{
		Message:  e.Message,
		Metadata: e.Metadata,
	}
}

package eventually

// Message represents any kind of information that can be carried around.
//
// Usually, a Message only contains a payload, but it could optionally
// include some metadata. (e.g. some debug identifiers)
type Message struct {
	Payload  interface{}
	Metadata Metadata
}

// Metadata contains some data related to a Message that are not functional
// for the Message itself, but instead functioning as supporting information
// to provide additional context.
type Metadata map[string]interface{}

// With returns a new Metadata reference holding the value addressed using
// the specified key.
func (m Metadata) With(key string, value interface{}) Metadata {
	if m == nil {
		m = make(Metadata, 1)
	}

	m[key] = value

	return m
}

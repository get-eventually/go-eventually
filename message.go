package eventually

type Message struct {
	Payload  interface{}
	Metadata Metadata
}

type Metadata map[string]interface{}

func (m Metadata) With(key string, value interface{}) Metadata {
	if m == nil {
		m = make(Metadata, 1)
	}

	m[key] = value

	return m
}

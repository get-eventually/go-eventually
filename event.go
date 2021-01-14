package eventually

const GlobalSequenceNumberKey = "Global-Sequence-Number"

type Event struct {
	Payload  interface{}
	Metadata Metadata
}

type Metadata map[string]interface{}

func (m Metadata) GlobalSequenceNumber() (int64, bool) {
	// NOTE: when unmarshaling the sequence number from JSON, it might happen
	// that the type of this field is float64.
	if f64, ok := m[GlobalSequenceNumberKey].(float64); ok {
		return int64(f64), true
	}

	i64, ok := m[GlobalSequenceNumberKey].(int64)
	return i64, ok
}

func (m *Metadata) WithGlobalSequenceNumber(v int64) {
	*m = m.With(GlobalSequenceNumberKey, v)
}

func (m Metadata) With(key string, value interface{}) Metadata {
	if m == nil {
		m = make(Metadata, 1)
	}

	m[key] = value

	return m
}

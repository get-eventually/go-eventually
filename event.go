package eventually

const GlobalSequenceNumberKey = "Global-Sequence-Number"

type Event struct {
	Payload  interface{}
	Metadata Metadata
}

type Metadata map[string]interface{}

func (m Metadata) GlobalSequenceNumber() (int64, bool) {
	v, ok := m[GlobalSequenceNumberKey].(int64)
	return v, ok
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

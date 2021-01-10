package eventually

type Event struct {
	Payload  interface{}
	Metadata Metadata
}

type Metadata map[string]interface{}

func (m Metadata) GlobalSequenceNumber() (int64, bool) {
	if m == nil {
		return 0, false
	}

	v, ok := m["Global-Sequence-Number"]
	if !ok {
		return 0, false
	}

	sequenceNumber, ok := v.(int64)

	return sequenceNumber, ok
}

func (m *Metadata) WithGlobalSequenceNumber(v int64) {
	if *m == nil {
		*m = make(Metadata, 1)
	}

	(*m)["Global-Sequence-Number"] = v
}

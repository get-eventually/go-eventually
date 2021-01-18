package eventually

const GlobalSequenceNumberKey = "Global-Sequence-Number"

type Event Message

func (evt Event) GlobalSequenceNumber() (int64, bool) {
	// NOTE: when unmarshaling the sequence number from JSON, it might happen
	// that the type of this field is float64.
	if f64, ok := evt.Metadata[GlobalSequenceNumberKey].(float64); ok {
		return int64(f64), true
	}

	i64, ok := evt.Metadata[GlobalSequenceNumberKey].(int64)
	return i64, ok
}

func (evt Event) WithGlobalSequenceNumber(v int64) Event {
	evt.Metadata = evt.Metadata.With(GlobalSequenceNumberKey, v)
	return evt
}

package eventually

// GlobalSequenceNumberKey is the Metadata key used to access the Global
// Sequence Number of an Event, which is the offset of the Event in the Event Store
// it comes from.
const GlobalSequenceNumberKey = "Global-Sequence-Number"

// Event is a Message representing some Domain information that has happened
// in the past, which is of vital information to the Domain itself.
//
// Event type names should be phrased in the past tense, to enforce the notion
// of "information happened in the past."
type Event Message

// GlobalSequenceNumber returns the global sequence number of the Event
// contained in the Metadata, if any.
//
// A boolean flag is returned alongside the sequence number value to indicate
// whether the sequence number was successfully found in the Metadata or not,
// thus making the int64 value returned meaningful.
func (evt Event) GlobalSequenceNumber() (int64, bool) {
	// NOTE: when unmarshaling the sequence number from JSON, it might happen
	// that the type of this field is float64.
	if f64, ok := evt.Metadata[GlobalSequenceNumberKey].(float64); ok {
		return int64(f64), true
	}

	i64, ok := evt.Metadata[GlobalSequenceNumberKey].(int64)
	return i64, ok
}

// WithGlobalSequenceNumber attaches the specified global sequence number
// value into the Event's Metadata.
func (evt Event) WithGlobalSequenceNumber(v int64) Event {
	evt.Metadata = evt.Metadata.With(GlobalSequenceNumberKey, v)
	return evt
}

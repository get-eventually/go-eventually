package internal

// IntPayload represents a generic integer message payload
// that can be used in test functions.
type IntPayload int64

// Name is the payload name of the IntPayload type.
func (IntPayload) Name() string { return "int_payload" }

// StringPayload represents a generic string message payload
// that can be used in test functions.
type StringPayload string

// Name is the payload name of the StringPayload type.
func (StringPayload) Name() string { return "string_payload" }

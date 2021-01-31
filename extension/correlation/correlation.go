package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go"
)

// Metadata keys used by the package.
const (
	EventIDKey       = "Event-Id"
	CorrelationIDKey = "Correlation-Id"
	CausationIDKey   = "Causation-Id"
)

type (
	correlationCtxKey struct{}
	causationCtxKey   struct{}
)

// WithCorrelationID adds the specified correlation id in the context,
// which will be used by the other extension components exposed by this package.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationCtxKey{}, id)
}

// WithCausationID adds the specified causation id in the context,
// which will be used by the other extension components exposed by this package.
func WithCausationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, causationCtxKey{}, id)
}

// Message extends an eventually.Message instance to fetch Correlation and
// Causation ids from the Message Metadata.
type Message eventually.Message

// CorrelationID returns the Correlation id from the Message Metadata, if found.
func (msg Message) CorrelationID() (string, bool) {
	v, ok := msg.Metadata[CorrelationIDKey]
	if !ok {
		return "", false
	}

	s, ok := v.(string)

	return s, ok
}

// CausationID returns the Causation id from the Message Metadata, if found.
func (msg Message) CausationID() (string, bool) {
	v, ok := msg.Metadata[CausationIDKey]
	if !ok {
		return "", false
	}

	s, ok := v.(string)

	return s, ok
}

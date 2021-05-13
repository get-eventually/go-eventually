package correlation

import (
	"context"

	"github.com/get-eventually/go-eventually"
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

// IDContext returns the Correlation id from the context,
// if it has been set using WithCorrelationID modifier.
func IDContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(correlationCtxKey{}).(string)
	return id, ok
}

// CausationIDContext returns the Causation id from the context,
// if it has been set using WithCausationID modifier.
func CausationIDContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(causationCtxKey{}).(string)
	return id, ok
}

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

// EventID returns the Message identifier from the Message Metadata, if found.
func (msg Message) EventID() (string, bool) {
	v, ok := msg.Metadata[EventIDKey]
	if !ok {
		return "", false
	}

	s, ok := v.(string)

	return s, ok
}

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

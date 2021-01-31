package correlation

import (
	"context"

	"github.com/eventually-rs/eventually-go"
)

const (
	EventIDKey       = "Event-Id"
	CorrelationIDKey = "Correlation-Id"
	CausationIDKey   = "Causation-Id"
)

type (
	correlationCtxKey struct{}
	causationCtxKey   struct{}
)

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationCtxKey{}, id)
}

func WithCausationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, causationCtxKey{}, id)
}

type Message eventually.Message

func (msg Message) CorrelationID() (string, bool) {
	v, ok := msg.Metadata[CorrelationIDKey]
	if !ok {
		return "", false
	}

	s, ok := v.(string)
	return s, ok
}

func (msg Message) CausationID() (string, bool) {
	v, ok := msg.Metadata[CausationIDKey]
	if !ok {
		return "", false
	}

	s, ok := v.(string)
	return s, ok
}

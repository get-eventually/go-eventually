package correlation

import "context"

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

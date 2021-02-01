package checkpoint

import "context"

// Checkpointer represents a component that contains the current state of
// progress of a named Subscription, so that the Subscription can be restored
// from where it left off, in case of application restart.
type Checkpointer interface {
	Read(ctx context.Context, key string) (int64, error)
	Write(ctx context.Context, key string, sequenceNumber int64) error
}

// NopCheckpointer is a Checkpointer implementation that always restarts
// from the beginning.
var NopCheckpointer = FixedCheckpointer{StartingFrom: 0}

// FixedCheckpointer is a Checkpointer implementation that always restarts
// from the specified starting point.
type FixedCheckpointer struct{ StartingFrom int64 }

func (fc FixedCheckpointer) Read(context.Context, string) (int64, error) { return fc.StartingFrom, nil }

func (fc FixedCheckpointer) Write(context.Context, string, int64) error { return nil }

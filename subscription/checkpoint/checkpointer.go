package checkpoint

import "context"

// Checkpointer represents a component that contains the current state of
// progress of a named Subscription, so that the Subscription can be restored
// from where it left off, in case of application restart.
type Checkpointer interface {
	Read(ctx context.Context, key string) (int64, error)
	Write(ctx context.Context, key string, sequenceNumber int64) error
}

// CheckpointerMock is a mock implementation of a checkpoint.Checkpointer.
type CheckpointerMock struct {
	// ReadFn is the function called on checkpoint.Checkpointer.Read.
	ReadFn func(ctx context.Context, key string) (int64, error)
	// WriteFn is the function called on checkpoint.Checkpointer.Write.
	WriteFn func(ctx context.Context, key string, sequenceNumber int64) error
}

// Read runs the checkpoint.CheckpointerMock.ReadFn function.
func (m CheckpointerMock) Read(ctx context.Context, key string) (int64, error) {
	return m.ReadFn(ctx, key)
}

// Write runs the checkpoint.CheckpointerMock.WriteFn function.
func (m CheckpointerMock) Write(ctx context.Context, key string, sequenceNumber int64) error {
	return m.WriteFn(ctx, key, sequenceNumber)
}

// NopCheckpointer is a Checkpointer implementation that always restarts
// from the beginning.
var NopCheckpointer = FixedCheckpointer{StartingFrom: 0}

// FixedCheckpointer is a Checkpointer implementation that always restarts
// from the specified starting point.
type FixedCheckpointer struct{ StartingFrom int64 }

func (fc FixedCheckpointer) Read(context.Context, string) (int64, error) { return fc.StartingFrom, nil }

func (fc FixedCheckpointer) Write(context.Context, string, int64) error { return nil }

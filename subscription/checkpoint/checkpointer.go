package checkpoint

import "context"

type Checkpointer interface {
	Read(ctx context.Context, key string) (int64, error)
	Write(ctx context.Context, key string, sequenceNumber int64) error
}

type NopCheckpointer struct{}

func (nc NopCheckpointer) Read(context.Context, string) (int64, error) { return 0, nil }

func (nc NopCheckpointer) Write(context.Context, string, int64) error { return nil }

type FixedCheckpointer struct{ StartingFrom int64 }

func (fc FixedCheckpointer) Read(context.Context, string) (int64, error) { return fc.StartingFrom, nil }

func (fc FixedCheckpointer) Write(context.Context, string, int64) error { return nil }

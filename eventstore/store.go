package eventstore

import (
	"context"

	"github.com/eventually-rs/eventually-go"
)

type EventStream chan<- Event

type Event struct {
	eventually.Event

	StreamType string
	StreamName string
	Version    int64
}

type Streamer interface {
	Stream(ctx context.Context, es EventStream, from int64) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, es EventStream) error
}

type Store interface {
	Streamer
	Subscriber

	Type(ctx context.Context, typ string) (Typed, error)
	Register(ctx context.Context, typ string, events map[string]interface{}) error
}

type Typed interface {
	Streamer
	Subscriber

	Instance(id string) Instanced
}

type Instanced interface {
	Streamer
	Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error)
}

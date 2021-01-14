package subscription

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"
)

var _ Subscription = Volatile{}

type Volatile struct {
	name       string
	subscriber eventstore.Subscriber
}

func NewVolatile(name string, subscriber eventstore.Subscriber) Volatile {
	return Volatile{
		name:       name,
		subscriber: subscriber,
	}
}

func (v Volatile) Name() string { return v.name }

func (v Volatile) Start(ctx context.Context, stream eventstore.EventStream) error {
	if err := v.subscriber.Subscribe(ctx, stream); err != nil {
		return fmt.Errorf("subscription.Volatile: subscriber exited with error: %w", err)
	}

	return nil
}

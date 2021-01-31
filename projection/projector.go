package projection

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/subscription"

	"golang.org/x/sync/errgroup"
)

const DefaultProjectorBufferSize = 128

type Projector struct {
	projection   Applier
	subscription subscription.Subscription
}

func NewProjector(projection Applier, subscription subscription.Subscription) *Projector {
	return &Projector{
		projection:   projection,
		subscription: subscription,
	}
}

func (p *Projector) Start(ctx context.Context) error {
	stream := make(chan eventstore.Event, DefaultProjectorBufferSize)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := p.subscription.Start(ctx, stream); err != nil {
			return fmt.Errorf("projector: subscription exited with error: %w", err)
		}

		return nil
	})

	for event := range stream {
		if err := p.projection.Apply(ctx, event); err != nil {
			return fmt.Errorf("projector: failed to apply event to projection: %w", err)
		}
	}

	return g.Wait()
}

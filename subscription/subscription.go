package subscription

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"

	"golang.org/x/sync/errgroup"
)

type Subscription interface {
	Name() string
	Start(context.Context, eventstore.EventStream) error
}

type Checkpointer interface {
	Get(context.Context, string) (int64, error)
	Store(context.Context, string, int64) error
}

var _ Subscription = CatchUp{}

type CatchUp struct {
	name            string
	startFrom       int64
	eventStreamer   eventstore.Streamer
	eventSubscriber eventstore.Subscriber
	checkpointer    Checkpointer
}

func NewCatchUp(
	ctx context.Context,
	name string,
	eventStreamer eventstore.Streamer,
	eventSubscriber eventstore.Subscriber,
	checkpointer Checkpointer,
) (CatchUp, error) {
	from, err := checkpointer.Get(ctx, name)
	if err != nil {
		return CatchUp{}, fmt.Errorf("subscription: failed to get checkpoint: %w", err)
	}

	return CatchUp{
		name:            name,
		startFrom:       from,
		eventStreamer:   eventStreamer,
		eventSubscriber: eventSubscriber,
		checkpointer:    checkpointer,
	}, nil
}

func (s CatchUp) Name() string { return s.name }

func (s CatchUp) Start(ctx context.Context, stream eventstore.EventStream) error {
	defer close(stream)

	streamed := make(chan eventstore.Event, len(stream))
	subscribed := make(chan eventstore.Event, len(stream))

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return s.eventSubscriber.Subscribe(ctx, subscribed) })
	group.Go(func() error { return s.eventStreamer.Stream(ctx, streamed, s.startFrom) })

	lastSequenceNumber := s.startFrom

	for event := range streamed {
		stream <- event

		sn, ok := event.Metadata.GlobalSequenceNumber()
		if ok {
			lastSequenceNumber = sn

			if err := s.checkpointer.Store(ctx, s.name, lastSequenceNumber); err != nil {
				return fmt.Errorf("subscription.CatchUp: failed to checkpoint: %w", err)
			}
		}
	}

	for event := range subscribed {
		sn, ok := event.Metadata.GlobalSequenceNumber()
		if ok && lastSequenceNumber >= sn {
			continue
		}

		stream <- event

		if ok {
			lastSequenceNumber = sn

			if err := s.checkpointer.Store(ctx, s.name, lastSequenceNumber); err != nil {
				return fmt.Errorf("subscription.CatchUp: failed to checkpoint: %w", err)
			}
		}
	}

	return group.Wait()
}

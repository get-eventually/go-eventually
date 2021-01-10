package subscription

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"

	"golang.org/x/sync/errgroup"
)

type Subscription interface {
	Name() string
	Start(context.Context, eventstore.EventStream) error
}

type CheckpointGetter interface {
	Get(context.Context, string) (int64, error)
}

type CheckpointStorer interface {
	Store(context.Context, string, int64) error
}

var _ Subscription = CatchUp{}

type CatchUp struct {
	name            string
	startFrom       int64
	eventStreamer   eventstore.Streamer
	eventSubscriber eventstore.Subscriber
	// checkpointStorer CheckpointStorer
}

func NewCatchUp(
	ctx context.Context,
	name string,
	eventStreamer eventstore.Streamer,
	eventSubscriber eventstore.Subscriber,
	// checkpointGetter CheckpointGetter,
	// checkpointStorer CheckpointStorer,
) (CatchUp, error) {
	// from, err := checkpointGetter.Get(ctx, name)
	// if err != nil {
	// 	return CatchUp{}, fmt.Errorf("subscription: failed to get checkpoint: %w", err)
	// }

	return CatchUp{
		name:            name,
		eventStreamer:   eventStreamer,
		eventSubscriber: eventSubscriber,
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
		if sn, ok := event.Metadata.GlobalSequenceNumber(); ok {
			lastSequenceNumber = sn
		}

		stream <- event
	}

	for event := range subscribed {
		sn, ok := event.Metadata.GlobalSequenceNumber()
		if ok && lastSequenceNumber >= sn {
			continue
		}

		stream <- event

		if ok {
			lastSequenceNumber = sn
		}
	}

	return group.Wait()
}

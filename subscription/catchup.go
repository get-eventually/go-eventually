package subscription

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go/eventstore"

	"golang.org/x/sync/errgroup"
)

var _ Subscription = CatchUp{}

type CatchUp struct {
	name            string
	startFrom       int64
	eventStreamer   eventstore.Streamer
	eventSubscriber eventstore.Subscriber
	checkpointer    Checkpointer
}

func NewCatchUp(
	name string,
	eventStreamer eventstore.Streamer,
	eventSubscriber eventstore.Subscriber,
	checkpointer Checkpointer,
) (CatchUp, error) {
	return CatchUp{
		name:            name,
		eventStreamer:   eventStreamer,
		eventSubscriber: eventSubscriber,
		checkpointer:    checkpointer,
	}, nil
}

func (s CatchUp) Name() string { return s.name }

func (s CatchUp) Start(ctx context.Context, stream eventstore.EventStream) error {
	lastSequenceNumber, err := s.checkpointer.Get(ctx, s.Name())
	if err != nil {
		return fmt.Errorf("subscription.CatchUp: failed to get checkpoint: %w", err)
	}

	defer close(stream)

	streamed := make(chan eventstore.Event, len(stream))
	subscribed := make(chan eventstore.Event, len(stream))

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return s.eventSubscriber.Subscribe(ctx, subscribed) })
	group.Go(func() error { return s.eventStreamer.Stream(ctx, streamed, s.startFrom) })

	if err := s.stream(ctx, &lastSequenceNumber, stream, streamed); err != nil {
		return err
	}

	if err := s.stream(ctx, &lastSequenceNumber, stream, subscribed); err != nil {
		return err
	}

	return group.Wait()
}

func (s CatchUp) stream(
	ctx context.Context,
	lastSequenceNumber *int64,
	stream eventstore.EventStream,
	source <-chan eventstore.Event,
) error {
	for event := range source {
		sn, ok := event.Metadata.GlobalSequenceNumber()
		if ok && *lastSequenceNumber >= sn {
			continue
		}

		stream <- event

		// Skip sequence number checkpointing if the event does not have a
		// global sequence number recorded.
		if !ok {
			continue
		}

		*lastSequenceNumber = sn

		if err := s.checkpointer.Store(ctx, s.Name(), *lastSequenceNumber); err != nil {
			return fmt.Errorf("subscription.CatchUp: failed to checkpoint: %w", err)
		}
	}

	return nil
}

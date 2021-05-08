package subscription_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
	"github.com/eventually-rs/eventually-go/internal"
	"github.com/eventually-rs/eventually-go/subscription"
	"github.com/eventually-rs/eventually-go/subscription/checkpoint"

	"github.com/stretchr/testify/assert"
)

func TestCatchUp(t *testing.T) {
	t.Run("catch-up subscriptions will receive events from before the subscription has started", func(t *testing.T) {
		streamID := eventstore.StreamID{
			Type: "my-type",
			Name: "my-instance",
		}

		ctx := context.Background()
		eventStore := inmemory.NewEventStore()

		events := []eventstore.Event{
			{
				StreamID: streamID,
				Version:  1,
				Event:    eventually.Event{Payload: internal.IntPayload(1)}.WithGlobalSequenceNumber(1),
			},
			{
				StreamID: streamID,
				Version:  2,
				Event:    eventually.Event{Payload: internal.IntPayload(2)}.WithGlobalSequenceNumber(2),
			},
			{
				StreamID: streamID,
				Version:  3,
				Event:    eventually.Event{Payload: internal.IntPayload(3)}.WithGlobalSequenceNumber(3),
			},
			{
				StreamID: streamID,
				Version:  4,
				Event:    eventually.Event{Payload: internal.IntPayload(4)}.WithGlobalSequenceNumber(4),
			},
		}

		// Append events before starting the subscription
		_, err := eventStore.Append(
			ctx,
			streamID,
			eventstore.VersionCheck(0),
			events[0].Event, events[1].Event,
		)

		if !assert.NoError(t, err) {
			return
		}

		// Start the subscription and listen to incoming events
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		catchupSubscription := subscription.CatchUp{
			SubscriptionName: t.Name(),
			Checkpointer:     checkpoint.NopCheckpointer,
			Target:           subscription.TargetStreamAll{}, // Using this just for coverage, lol
			EventStore:       eventStore,
		}

		wg := new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			defer cancel()

			wg.Wait()

			//nolint:govet // The shadowing of this variable is meant to avoid updating the one
			//                outside the goroutine function declaration.
			_, err := eventStore.Append(
				ctx,
				streamID,
				eventstore.VersionCheck(2),
				events[2].Event, events[3].Event,
			)

			assert.NoError(t, err)

			// NOTE: this is bad, I know, and it makes the test kinda unreliable,
			// but in order to ensure that the subscription has consumed
			// all the events committed before closing the context we gotta wait
			// a little bit...
			<-time.After(100 * time.Millisecond)
		}()

		received, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			// This kinda helps with starting the Subscription first,
			// then wake up the WaitGroup, which will unlock the write goroutine.
			go func() { wg.Done() }()
			return catchupSubscription.Start(ctx, es)
		})

		assert.True(t, errors.Is(err, context.Canceled), "err", err)
		assert.Equal(t, events, received)
	})
}

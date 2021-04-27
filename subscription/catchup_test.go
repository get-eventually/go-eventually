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
		const (
			myType     = "my-type"
			myInstance = "my-instance"
		)

		ctx := context.Background()
		eventStore := inmemory.NewEventStore()

		if err := eventStore.Register(ctx, myType, nil); !assert.NoError(t, err) {
			return
		}

		typedEventStore, err := eventStore.Type(ctx, myType)
		if !assert.NoError(t, err) {
			return
		}

		events := []eventstore.Event{
			{
				StreamType: myType,
				StreamName: myInstance,
				Version:    1,
				Event:      eventually.Event{Payload: internal.IntPayload(1)}.WithGlobalSequenceNumber(1),
			},
			{
				StreamType: myType,
				StreamName: myInstance,
				Version:    2,
				Event:      eventually.Event{Payload: internal.IntPayload(2)}.WithGlobalSequenceNumber(2),
			},
			{
				StreamType: myType,
				StreamName: myInstance,
				Version:    3,
				Event:      eventually.Event{Payload: internal.IntPayload(3)}.WithGlobalSequenceNumber(3),
			},
			{
				StreamType: myType,
				StreamName: myInstance,
				Version:    4,
				Event:      eventually.Event{Payload: internal.IntPayload(4)}.WithGlobalSequenceNumber(4),
			},
		}

		// Append events before starting the subscription
		_, err = typedEventStore.
			Instance(myInstance).
			Append(ctx, 0, events[0].Event, events[1].Event)

		if !assert.NoError(t, err) {
			return
		}

		// Start the subscription and listen to incoming events
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		catchupSubscription := subscription.CatchUp{
			SubscriptionName: t.Name(),
			Checkpointer:     checkpoint.NopCheckpointer,
			EventStore:       typedEventStore,
		}

		wg := new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			defer cancel()

			wg.Wait()

			_, err = typedEventStore.
				Instance(myInstance).
				Append(ctx, 2, events[2].Event, events[3].Event)

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

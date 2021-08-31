package projection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/internal"
	"github.com/get-eventually/go-eventually/logger"
	"github.com/get-eventually/go-eventually/projection"
	"github.com/get-eventually/go-eventually/subscription"
	"github.com/get-eventually/go-eventually/subscription/checkpoint"
)

var streamID = stream.ID{
	Type: "my-type",
	Name: "my-instance",
}

var expectedEvents = []eventstore.Event{
	{
		Stream:         streamID,
		Version:        1,
		SequenceNumber: 1,
		Event:          eventually.Event{Payload: internal.IntPayload(1)},
	},
	{
		Stream:         streamID,
		Version:        2,
		SequenceNumber: 2,
		Event:          eventually.Event{Payload: internal.IntPayload(2)},
	},
	{
		Stream:         streamID,
		Version:        3,
		SequenceNumber: 3,
		Event:          eventually.Event{Payload: internal.IntPayload(3)},
	},
}

func TestRunner(t *testing.T) {
	t.Run("it works", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		eventStore := inmemory.NewEventStore()

		// Create a new subscription to listen events from the event store
		testSubscription := &subscription.CatchUp{
			SubscriptionName: "test-subscription",
			Target:           stream.All{},
			EventStore:       eventStore,
			Checkpointer:     checkpoint.NopCheckpointer,
			PullEvery:        10 * time.Millisecond,
			MaxInterval:      50 * time.Millisecond,
			Logger:           logger.Nop{},
		}

		// The target applier function triggers a side effect that projects all received
		// events onto this slice.
		var received []eventstore.Event

		receivedMx := new(sync.RWMutex)

		applier := projection.ApplierFunc(func(_ context.Context, event eventstore.Event) error {
			receivedMx.Lock()
			defer receivedMx.Unlock()

			received = append(received, event)
			return nil
		})

		runner := projection.Runner{
			Applier:      applier,
			Subscription: testSubscription,
		}

		group, ctx := errgroup.WithContext(ctx)
		group.Go(func() error {
			return runner.Run(ctx)
		})

		for _, event := range expectedEvents {
			_, err := eventStore.Append(
				ctx,
				streamID,
				eventstore.VersionCheck(event.Version-1),
				event.Event,
			)

			if !assert.NoError(t, err) {
				return
			}
		}

		// Some active waiting to make sure the Runner has drained the subscription
		// event stream and updated the Applier successfully.
		// Some active waiting to make sure the Runner has drained the subscription
		// event stream and updated the Applier successfully.
		attempts := 0
		ticker := time.NewTicker(100 * time.Millisecond)

		for range ticker.C {
			receivedMx.RLock()

			if len(received) > 0 || attempts > 10 {
				return
			}

			attempts++

			receivedMx.RUnlock()
		}

		cancel()

		if !assert.ErrorIs(t, group.Wait(), context.Canceled) {
			return
		}

		assert.Equal(t, expectedEvents, received)
	})

	t.Run("does not checkpoint when using projection.DoNotCheckpoint", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		eventStore := inmemory.NewEventStore()
		_, err := eventStore.Append(ctx, streamID, eventstore.VersionCheck(0), eventstore.MapEvents(expectedEvents)...)
		require.NoError(t, err)

		// Create a new subscription to listen events from the event store
		testSubscription := &subscription.CatchUp{
			SubscriptionName: "test-subscription",
			Target:           stream.All{},
			EventStore:       eventStore,
			PullEvery:        10 * time.Millisecond,
			MaxInterval:      50 * time.Millisecond,
			Logger:           logger.Nop{},
			Checkpointer: checkpoint.CheckpointerMock{
				ReadFn: func(ctx context.Context, key string) (int64, error) {
					return 0, nil
				},
				WriteFn: func(ctx context.Context, key string, sequenceNumber int64) error {
					assert.Fail(t, "write should not be called", "key", key, "sequenceNumber", sequenceNumber)

					return nil
				},
			},
		}

		// The target applier function triggers a side effect that projects all received
		// events onto this slice.
		var received []eventstore.Event

		receivedMx := new(sync.RWMutex)
		applier := projection.ApplierFunc(func(ctx context.Context, event eventstore.Event) error {
			receivedMx.Lock()
			defer receivedMx.Unlock()

			// Avoid checkpointing messages.
			projection.DoNotCheckpoint(ctx)

			received = append(received, event)

			if len(received) == len(expectedEvents) {
				// Cancel the context once all events have arrived.
				cancel()
			}

			return nil
		})

		runner := projection.Runner{
			Applier:      applier,
			Subscription: testSubscription,
		}

		assert.ErrorIs(t, runner.Run(ctx), context.Canceled)
	})
}

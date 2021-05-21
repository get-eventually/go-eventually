package subscription_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/internal"
	"github.com/get-eventually/go-eventually/subscription"
)

func TestVolatile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamID := eventstore.StreamID{
		Type: "my-type",
		Name: "my-instance",
	}

	eventStore := inmemory.NewEventStore()

	volatileSubscription := subscription.Volatile{
		SubscriptionName: "test-volatile-subscription",
		Target:           subscription.TargetStreamType{Type: streamID.Type},
		EventStore:       eventStore,
	}

	_, err := eventStore.Append(
		ctx,
		streamID,
		eventstore.VersionCheck(0),
		eventually.Event{Payload: internal.StringPayload("test-event-should-not-be-received")},
	)

	if !assert.NoError(t, err) {
		return
	}

	expectedEvents := []eventstore.Event{
		{
			Stream:         streamID,
			Version:        2,
			SequenceNumber: 2,
			Event: eventually.Event{
				Payload: internal.StringPayload("test-event-should-be-received-0"),
			},
		},
		{
			Stream:         streamID,
			Version:        3,
			SequenceNumber: 3,
			Event: eventually.Event{
				Payload: internal.StringPayload("test-event-should-be-received-1"),
			},
		},
		{
			Stream:         streamID,
			Version:        4,
			SequenceNumber: 4,
			Event: eventually.Event{
				Payload: internal.StringPayload("test-event-should-be-received-2"),
			},
		},
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		defer cancel()
		wg.Wait()

		for i := 0; i < 3; i++ {
			_, err = eventStore.Append(
				ctx,
				streamID,
				eventstore.VersionCheck(i+1),
				eventually.Event{
					Payload: internal.StringPayload(fmt.Sprintf("test-event-should-be-received-%d", i)),
				},
			)

			if !assert.NoError(t, err) {
				return
			}
		}
	}()

	events, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
		// This kinda helps with starting the Subscription first,
		// then wake up the WaitGroup, which will unlock the write goroutine.
		go func() { wg.Done() }()
		return volatileSubscription.Start(ctx, es)
	})

	assert.True(t, errors.Is(err, context.Canceled), "err", err)
	assert.Equal(t, expectedEvents, events)
}

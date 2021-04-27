package subscription_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
	"github.com/eventually-rs/eventually-go/internal"
	"github.com/eventually-rs/eventually-go/subscription"

	"github.com/stretchr/testify/assert"
)

func TestVolatile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		typeName     = "my-type"
		instanceName = "my-instance"
	)

	eventStore := inmemory.NewEventStore()

	if err := eventStore.Register(ctx, typeName, nil); !assert.NoError(t, err) {
		return
	}

	typedEventStore, err := eventStore.Type(ctx, typeName)
	if !assert.NoError(t, err) {
		return
	}

	volatileSubscription := subscription.Volatile{
		SubscriptionName: "test-volatile-subscription",
		EventSubscriber:  eventStore,
	}

	_, err = typedEventStore.
		Instance(instanceName).
		Append(ctx, 0, eventually.Event{Payload: internal.StringPayload("test-event-should-not-be-received")})

	if !assert.NoError(t, err) {
		return
	}

	expectedEvents := []eventstore.Event{
		{
			StreamType: typeName,
			StreamName: instanceName,
			Version:    2,
			Event: eventually.
				Event{Payload: internal.StringPayload("test-event-should-be-received-0")}.
				WithGlobalSequenceNumber(2),
		},
		{
			StreamType: typeName,
			StreamName: instanceName,
			Version:    3,
			Event: eventually.
				Event{Payload: internal.StringPayload("test-event-should-be-received-1")}.
				WithGlobalSequenceNumber(3),
		},
		{
			StreamType: typeName,
			StreamName: instanceName,
			Version:    4,
			Event: eventually.
				Event{Payload: internal.StringPayload("test-event-should-be-received-2")}.
				WithGlobalSequenceNumber(4),
		},
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		defer cancel()
		wg.Wait()

		for i := 0; i < 3; i++ {
			_, err = typedEventStore.
				Instance(instanceName).
				Append(ctx, int64(i+1), eventually.Event{
					Payload: internal.StringPayload(fmt.Sprintf("test-event-should-be-received-%d", i)),
				})

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

package projection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
	"github.com/eventually-rs/eventually-go/projection"
	"github.com/eventually-rs/eventually-go/subscription"
	"github.com/eventually-rs/eventually-go/subscription/checkpoint"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

const (
	streamType     = "runner-type"
	streamInstance = "runner-instance"
)

var expectedEvents = []eventstore.Event{
	{
		StreamType: streamType,
		StreamName: streamInstance,
		Version:    1,
		Event:      eventually.Event{Payload: 1}.WithGlobalSequenceNumber(1),
	},
	{
		StreamType: streamType,
		StreamName: streamInstance,
		Version:    2,
		Event:      eventually.Event{Payload: 2}.WithGlobalSequenceNumber(2),
	},
	{
		StreamType: streamType,
		StreamName: streamInstance,
		Version:    3,
		Event:      eventually.Event{Payload: 3}.WithGlobalSequenceNumber(3),
	},
}

func TestRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventStore := inmemory.NewEventStore()

	// Need to register the stream type before sending messages
	if !assert.NoError(t, eventStore.Register(ctx, streamType, nil)) {
		return
	}

	typedStore, err := eventStore.Type(ctx, streamType)
	if !assert.NoError(t, err) {
		return
	}

	// Create a new subscription to listen events from the event store
	testSubscription := subscription.CatchUp{
		SubscriptionName: "test-subscription",
		EventStore:       typedStore,
		Checkpointer:     checkpoint.NopCheckpointer,
	}

	// The target applier function triggers a side effect that projects all received
	// events onto this slice.
	var received []eventstore.Event

	applier := projection.ApplierFunc(func(_ context.Context, event eventstore.Event) error {
		received = append(received, event)
		return nil
	})

	runner := projection.Runner{
		Applier:      applier,
		Subscription: testSubscription,
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		go func() { wg.Done() }()
		return runner.Run(ctx)
	})

	// This is to ensure to append events only after the subscription has been
	// started from the Runner.
	wg.Wait()

	for _, event := range expectedEvents {
		_, err := typedStore.
			Instance(streamInstance).
			Append(ctx, event.Version-1, event.Event)

		if !assert.NoError(t, err) {
			return
		}
	}

	// Some active waiting to make sure the Runner has drained the subscription
	// event stream and updated the Applier successfully.
	<-time.After(100 * time.Millisecond)
	cancel()

	if !assert.NoError(t, group.Wait()) {
		return
	}

	assert.Equal(t, expectedEvents, received)
}

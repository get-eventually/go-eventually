package inmemory_test

import (
	"context"
	"testing"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/inmemory"
	"github.com/eventually-rs/eventually-go/subscription"

	"github.com/stretchr/testify/assert"
)

func TestTypedSubscribe(t *testing.T) {
	store := inmemory.NewEventStore()
	store.Register("mytype")

	mytypeStore, _ := store.Type("mytype")
	stream := make(chan eventstore.Event, 1)
	go mytypeStore.Subscribe(context.Background(), stream)

	go func() {
		for event := range stream {
			t.Logf("Received event on mytype: %#v", event)
		}
	}()

	v, err := mytypeStore.
		Instance("test").
		Append(context.Background(), -1, eventually.Event{Payload: "hello"})

	t.Logf("Version: %d, err: %v", v, err)

	<-time.After(5 * time.Second)
}

func TestTypedSubscribeCloseContext(t *testing.T) {
	store := inmemory.NewEventStore()
	store.Register("mytype")

	ctx, cancel := context.WithCancel(context.Background())

	mytypeStore, _ := store.Type("mytype")
	stream := make(chan eventstore.Event, 1)
	go mytypeStore.Subscribe(ctx, stream)
	cancel()

	go func() {
		for event := range stream {
			t.Logf("Received event on mytype: %#v", event)
		}
	}()

	v, err := mytypeStore.
		Instance("test").
		Append(context.Background(), -1, eventually.Event{Payload: "hello"})

	e, ok := <-stream

	t.Logf("Version: %d, err: %v", v, err)
	t.Logf("Channel: %#v, closed: %v", e, !ok)
}

func TestStreamAll(t *testing.T) {
	store := inmemory.NewEventStore()

	store.Register("mytype")
	store.Register("myothertype")

	mytypeStore, _ := store.Type("mytype")
	myothertypeStore, _ := store.Type("myothertype")

	for i := 0; i < 10; i++ {
		mytypeStore.
			Instance("test").
			Append(context.Background(), int64(i), eventually.Event{Payload: i})

		myothertypeStore.
			Instance("test").
			Append(context.Background(), int64(i), eventually.Event{Payload: i})
	}

	stream := make(chan eventstore.Event, 1)
	go store.Stream(context.Background(), stream, 0)

	for event := range stream {
		t.Logf("Event received: %#v", event)
	}
}

func TestCatchUpSubscription(t *testing.T) {
	store := inmemory.NewEventStore()

	store.Register("mytype")
	store.Register("myothertype")

	mytypeStore, _ := store.Type("mytype")
	myothertypeStore, _ := store.Type("myothertype")

	subscription, err := subscription.NewCatchUp(
		context.TODO(),
		"test-subscription",
		mytypeStore,
		mytypeStore,
	)

	if !assert.NoError(t, err) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := make(chan eventstore.Event, 1)
	go subscription.Start(ctx, stream)

	go func() {
		defer cancel()

		for i := 0; i < 10; i++ {
			_, err := mytypeStore.Instance("test").Append(ctx, int64(i), eventually.Event{Payload: i})
			assert.NoError(t, err)

			_, err = myothertypeStore.Instance("test").Append(ctx, int64(i), eventually.Event{Payload: i})
			assert.NoError(t, err)
		}
	}()

	for event := range stream {
		t.Logf("Received event from subscription: %#v", event)
	}
}

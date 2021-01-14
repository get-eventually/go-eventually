package inmemory_test

import (
	"context"
	"testing"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"
	"github.com/eventually-rs/eventually-go/subscription"

	"github.com/stretchr/testify/assert"
)

func TestTypedSubscribe(t *testing.T) {
	ctx := context.Background()

	store := inmemory.NewEventStore()
	// We don't need events registration.
	if err := store.Register(ctx, "mytype", nil); !assert.NoError(t, err) {
		return
	}

	mytypeStore, err := store.Type(ctx, "mytype")
	if !assert.NoError(t, err) {
		return
	}

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := inmemory.NewEventStore()
	if err := store.Register(ctx, "mytype", nil); !assert.NoError(t, err) {
		return
	}

	mytypeStore, err := store.Type(ctx, "mytype")
	if !assert.NoError(t, err) {
		return
	}

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
	ctx := context.Background()
	store := inmemory.NewEventStore()

	if err := store.Register(ctx, "mytype", nil); !assert.NoError(t, err) {
		return
	}

	if err := store.Register(ctx, "myothertype", nil); !assert.NoError(t, err) {
		return
	}

	mytypeStore, err := store.Type(ctx, "mytype")
	if !assert.NoError(t, err) {
		return
	}

	myothertypeStore, err := store.Type(ctx, "myothertype")
	if !assert.NoError(t, err) {
		return
	}

	for i := 0; i < 10; i++ {
		mytypeStore.
			Instance("test").
			Append(ctx, int64(i), eventually.Event{Payload: i})

		myothertypeStore.
			Instance("test").
			Append(ctx, int64(i), eventually.Event{Payload: i})
	}

	stream := make(chan eventstore.Event, 1)
	go store.Stream(ctx, stream, 0)

	for event := range stream {
		t.Logf("Event received: %#v", event)
	}
}

func TestCatchUpSubscription(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := inmemory.NewEventStore()

	if err := store.Register(ctx, "mytype", nil); !assert.NoError(t, err) {
		return
	}

	if err := store.Register(ctx, "myothertype", nil); !assert.NoError(t, err) {
		return
	}

	mytypeStore, err := store.Type(ctx, "mytype")
	if !assert.NoError(t, err) {
		return
	}

	myothertypeStore, err := store.Type(ctx, "myothertype")
	if !assert.NoError(t, err) {
		return
	}

	subscription, err := subscription.NewCatchUp(
		"test-subscription",
		mytypeStore,
		mytypeStore,
		subscription.NopCheckpointer{},
	)

	if !assert.NoError(t, err) {
		return
	}

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

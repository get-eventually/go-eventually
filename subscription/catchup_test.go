package subscription_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/extension/zaplogger"
	"github.com/get-eventually/go-eventually/internal"
	"github.com/get-eventually/go-eventually/subscription"
	"github.com/get-eventually/go-eventually/subscription/checkpoint"
)

func TestCatchUp(t *testing.T) {
	s := new(CatchUpSuite)

	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	s.makeSubscription = func(store eventstore.Store) subscription.Subscription {
		return &subscription.CatchUp{
			SubscriptionName: t.Name(),
			Checkpointer:     checkpoint.NopCheckpointer,
			Target:           subscription.TargetStreamAll{},
			Logger:           zaplogger.Wrap(logger),
			EventStore:       store,
			PullEvery:        10 * time.Millisecond,
			MaxInterval:      50 * time.Millisecond,
		}
	}

	suite.Run(t, s)
}

type CatchUpSuite struct {
	suite.Suite

	makeSubscription func(eventstore.Store) subscription.Subscription
	eventStore       eventstore.Store
	subscription     subscription.Subscription
}

func (s *CatchUpSuite) SetupTest() {
	s.eventStore = inmemory.NewEventStore()
	s.subscription = s.makeSubscription(s.eventStore)
}

func (s *CatchUpSuite) TestCatchUp() {
	s.Run("catch-up subscriptions will receive events from before the subscription has started", func() {
		streamID := eventstore.StreamID{
			Type: "my-type",
			Name: "my-instance",
		}

		t := s.T()
		ctx := context.Background()

		events := []eventstore.Event{
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
			{
				Stream:         streamID,
				Version:        4,
				SequenceNumber: 4,
				Event:          eventually.Event{Payload: internal.IntPayload(4)},
			},
		}

		// Append events before starting the subscription
		_, err := s.eventStore.Append(
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

		wg := new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			defer cancel()

			wg.Wait()

			//nolint:govet // The shadowing of this variable is meant to avoid updating the one
			//                outside the goroutine function declaration.
			_, err := s.eventStore.Append(
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
			<-time.After(800 * time.Millisecond)
		}()

		received, err := eventstore.StreamToSlice(ctx, func(ctx context.Context, es eventstore.EventStream) error {
			// This kinda helps with starting the Subscription first,
			// then wake up the WaitGroup, which will unlock the write goroutine.
			go func() { wg.Done() }()
			return s.subscription.Start(ctx, es)
		})

		assert.True(t, errors.Is(err, context.Canceled), "err", err)
		assert.Equal(t, events, received)
	})
}

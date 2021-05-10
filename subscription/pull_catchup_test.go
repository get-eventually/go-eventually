package subscription_test

import (
	"testing"
	"time"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/subscription"
	"github.com/eventually-rs/eventually-go/subscription/checkpoint"

	"github.com/stretchr/testify/suite"
)

func TestPullCatchUp(t *testing.T) {
	s := new(CatchUpSuite)

	s.makeSubscription = func(store eventstore.Store) subscription.Subscription {
		return &subscription.PullCatchUp{
			SubscriptionName: t.Name(),
			Checkpointer:     checkpoint.NopCheckpointer,
			Target:           subscription.TargetStreamAll{},
			EventStore:       store,
			PullEvery:        10 * time.Millisecond,
			MaxInterval:      50 * time.Millisecond,
		}
	}

	suite.Run(t, s)
}

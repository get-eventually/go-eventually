package subscription_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/extension/zaplogger"
	"github.com/get-eventually/go-eventually/subscription"
	"github.com/get-eventually/go-eventually/subscription/checkpoint"
)

func TestPullCatchUp(t *testing.T) {
	s := new(CatchUpSuite)

	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	s.makeSubscription = func(store eventstore.Store) subscription.Subscription {
		return &subscription.PullCatchUp{
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

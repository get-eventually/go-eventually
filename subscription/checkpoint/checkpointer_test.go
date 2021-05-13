package checkpoint_test

import (
	"context"
	"testing"

	"github.com/get-eventually/go-eventually/subscription/checkpoint"

	"github.com/stretchr/testify/assert"
)

const subscriptionName = "test-subscription"

func TestNopCheckpointer(t *testing.T) {
	ctx := context.Background()

	seqNum, err := checkpoint.NopCheckpointer.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Zero(t, seqNum)

	err = checkpoint.NopCheckpointer.Write(ctx, subscriptionName, 1200)
	assert.NoError(t, err)

	seqNum, err = checkpoint.NopCheckpointer.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Zero(t, seqNum)
}

func TestFixedCheckpointer(t *testing.T) {
	ctx := context.Background()
	start := int64(100)
	checkpointer := checkpoint.FixedCheckpointer{StartingFrom: start}

	seqNum, err := checkpointer.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Equal(t, start, seqNum)

	err = checkpoint.NopCheckpointer.Write(ctx, subscriptionName, 1200)
	assert.NoError(t, err)

	seqNum, err = checkpointer.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Equal(t, start, seqNum)
}

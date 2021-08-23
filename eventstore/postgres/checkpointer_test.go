package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/get-eventually/go-eventually/eventstore/postgres"
	"github.com/get-eventually/go-eventually/extension/zaplogger"
)

func TestCheckpointer(t *testing.T) {
	store := obtainEventStore(t)
	defer func() { assert.NoError(t, store.Close()) }()

	db, err := openDB()
	require.NoError(t, err)

	log := zaplogger.Wrap(zap.NewNop())
	ctx := context.Background()

	checkpointer := postgres.Checkpointer{
		DB:     db,
		Logger: log,
	}

	const subscriptionName = "test-subscription"

	seqNum, err := checkpointer.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Zero(t, seqNum)

	newSeqNum := int64(1200)
	err = checkpointer.Write(ctx, subscriptionName, newSeqNum)
	assert.NoError(t, err)

	seqNum, err = checkpointer.Read(ctx, subscriptionName)
	assert.NoError(t, err)
	assert.Equal(t, newSeqNum, seqNum)
}

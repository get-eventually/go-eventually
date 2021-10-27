package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/extension/postgres"
)

func TestCheckpointer(t *testing.T) {
	db, _, _ := obtainEventStore(t)
	defer func() { assert.NoError(t, db.Close()) }()

	ctx := context.Background()

	checkpointer := postgres.Checkpointer{
		DB: db,
		Logger: eventually.Logger{
			Debugf: t.Logf,
			Infof:  t.Logf,
			Errorf: t.Errorf,
		},
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

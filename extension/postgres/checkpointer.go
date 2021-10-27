package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/get-eventually/go-eventually"
)

// Checkpointer is a checkpoint.Checkpointer implementation using Postgres
// as a storage backend.
type Checkpointer struct {
	DB     *sql.DB
	Logger eventually.Logger
}

// Read reads the latest checkpointed sequence number of the subscription specified.
func (c Checkpointer) Read(ctx context.Context, subscriptionName string) (int64, error) {
	row := c.DB.QueryRowContext(
		ctx,
		"SELECT get_or_create_subscription_checkpoint($1)",
		subscriptionName,
	)

	var lastSequenceNumber int64
	if err := row.Scan(&lastSequenceNumber); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to read subscription checkpoint: %w", err)
	}

	return lastSequenceNumber, nil
}

// Write checkpoints the sequence number value provided for the specified subscription.
func (c Checkpointer) Write(ctx context.Context, subscriptionName string, sequenceNumber int64) error {
	_, err := c.DB.ExecContext(
		ctx,
		`UPDATE subscriptions_checkpoints
		SET last_sequence_number = $1
		WHERE subscription_id = $2`,
		sequenceNumber,
		subscriptionName,
	)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to write subscription checkpoint: %w", err)
	}

	return nil
}

package internal

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxBeginner represents a pgx-related component that can initiate transactions.
type TxBeginner interface {
	BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error)
}

// RunTransaction runs a critical data change path in a transaction,
// seamlessly handling the transaction lifecycle (begin, commit, rollback).
func RunTransaction(
	ctx context.Context,
	db TxBeginner,
	options pgx.TxOptions, //nolint:gocritic // The pgx API uses value semantics, will do the same here.
	do func(ctx context.Context, tx pgx.Tx) error,
) (err error) {
	withContext := func(msg string, err error) error {
		return fmt.Errorf("%s, %w", msg, err)
	}

	tx, err := db.BeginTx(ctx, options)
	if err != nil {
		return withContext("failed to begin transaction", err)
	}

	defer func() {
		if err == nil {
			return
		}

		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			err = fmt.Errorf("failed to rollback transaction, %w (caused by: %w)", rollbackErr, err)
		}
	}()

	if err := do(ctx, tx); err != nil {
		return withContext("failed to perform transaction", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return withContext("failed to commit transaction", err)
	}

	return nil
}

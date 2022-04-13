package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	Conn           *pgx.Conn
	AggregateType  aggregate.Type[ID, T]
	AggregateSerde serde.Bytes[T]
	MessageSerde   serde.Bytes[message.Message]
}

func (repo AggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	var zeroValue T

	row := repo.Conn.QueryRow(
		ctx,
		`SELECT version, state
		FROM aggregates
		WHERE aggregate_id = $1 AND "type" = $2`,
		id.String(), repo.AggregateType.Name,
	)

	var (
		v     version.Version
		state []byte
	)

	err := row.Scan(&v, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return zeroValue, aggregate.ErrRootNotFound
	}
	if err != nil {
		return zeroValue, fmt.Errorf(
			"postgres.AggregateRepository.Get: failed to fetch aggregate state from database: %w",
			err,
		)
	}

	root, err := aggregate.RehydrateFromState[ID, T, []byte](v, state, repo.AggregateSerde)
	if err != nil {
		return zeroValue, fmt.Errorf(
			"postgres.AggregateRepository.Get: failed to deserialize state into aggregate root object: %w",
			err,
		)
	}

	return root, nil
}

func (repo AggregateRepository[ID, T]) saveErr(msg string, args ...any) error {
	return fmt.Errorf("postgres.AggregateRepository.Save: "+msg, args...)
}

func (repo AggregateRepository[ID, T]) saveAggregateState(
	ctx context.Context,
	tx pgx.Tx,
	expectedVersion version.Version,
	root T,
) error {
	state, err := repo.AggregateSerde.Serialize(root)
	if err != nil {
		return repo.saveErr("failed to serialize aggregate root into wire format, %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`CALL upsert_aggregate($1::TEXT, $2::TEXT, $3::INTEGER, $4::INTEGER, $5::BYTEA)`,
		root.AggregateID().String(), repo.AggregateType.Name, expectedVersion, root.Version(), state,
	)

	if vc, ok := isVersionConflictError(err); ok {
		return repo.saveErr("failed to save new aggregate state, %w", vc)
	}

	if err != nil {
		return repo.saveErr("failed to save new aggregate state, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) Save(ctx context.Context, root T) error {
	conn := repo.Conn

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	})

	if err != nil {
		return repo.saveErr("failed to open db transaction, %w", err)
	}

	defer func() {
		// NOTE: should not have effect if the transaction has been committed
		_ = tx.Rollback(ctx)
	}()

	eventsToCommit := root.FlushRecordedEvents()
	expectedRootVersion := root.Version() - version.Version(len(eventsToCommit))

	if err := repo.saveAggregateState(ctx, tx, expectedRootVersion, root); err != nil {
		return err
	}

	eventStreamID := event.StreamID(root.AggregateID().String())

	err = appendDomainEvents(ctx, tx, repo.MessageSerde, eventStreamID, root.Version(), eventsToCommit...)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.saveErr("failed to commit transaction, %w", err)
	}

	return nil
}

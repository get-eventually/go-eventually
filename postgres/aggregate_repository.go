package eventuallypostgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

// AggregateRepository implements the aggregate.Repository interface
// for PostgreSQL databases.
//
// This implementation uses the "aggregates" table in the database
// as its main operational table. At the same time, it also writes
// to both "events" and "event_streams" to append the Domain events
// recorded by Aggregate Roots. These updates are performed within the same transaction.
type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	Conn           *pgxpool.Pool
	AggregateType  aggregate.Type[ID, T]
	AggregateSerde serde.Bytes[T]
	MessageSerde   serde.Bytes[message.Message]
}

// Get returns the aggregate.Root instance specified by the provided id.
// Returns aggregate.ErrRootNotFound if the Aggregate Root doesn't exist.
func (repo AggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	return repo.get(ctx, repo.Conn, id)
}

func (repo AggregateRepository[ID, T]) get(
	ctx context.Context,
	tx interface {
		QueryRow(context.Context, string, ...interface{}) pgx.Row
	},
	id ID,
) (T, error) {
	var zeroValue T

	row := tx.QueryRow(
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

	if err := row.Scan(&v, &state); errors.Is(err, pgx.ErrNoRows) {
		return zeroValue, aggregate.ErrRootNotFound
	} else if err != nil {
		return zeroValue, fmt.Errorf(
			"eventuallypostgres.AggregateRepository.Get: failed to fetch aggregate state from database: %w",
			err,
		)
	}

	root, err := aggregate.RehydrateFromState[ID, T, []byte](v, state, repo.AggregateSerde)
	if err != nil {
		return zeroValue, fmt.Errorf(
			"eventuallypostgres.AggregateRepository.Get: failed to deserialize state into aggregate root object: %w",
			err,
		)
	}

	return root, nil
}

// Save saves the new state of the provided aggregate.Root instance.
func (repo AggregateRepository[ID, T]) Save(ctx context.Context, root T) error {
	conn := repo.Conn

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.Serializable,
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
	eventStreamID := event.StreamID(root.AggregateID().String())

	newEventStreamVersion, err := appendDomainEvents(
		ctx, tx,
		repo.MessageSerde,
		eventStreamID,
		version.CheckExact(expectedRootVersion),
		eventsToCommit...,
	)
	if err != nil {
		return err
	}

	if newEventStreamVersion != root.Version() {
		return repo.saveErr("version mismatch between event stream and aggregate", version.ConflictError{
			Expected: newEventStreamVersion,
			Actual:   root.Version(),
		})
	}

	if err := repo.saveAggregateState(ctx, tx, eventStreamID, root); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.saveErr("failed to commit transaction, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) saveAggregateState(
	ctx context.Context,
	tx pgx.Tx,
	id event.StreamID,
	root T,
) error {
	state, err := repo.AggregateSerde.Serialize(root)
	if err != nil {
		return repo.saveErr("failed to serialize aggregate root into wire format, %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO aggregates (aggregate_id, "type", "version", "state")
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (aggregate_id) DO
		UPDATE SET "version" = $3, "state" = $4`,
		id, repo.AggregateType.Name, root.Version(), state,
	); err != nil {
		return repo.saveErr("failed to save new aggregate state, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) saveErr(msg string, args ...any) error {
	return fmt.Errorf("eventuallypostgres.AggregateRepository: "+msg, args...)
}

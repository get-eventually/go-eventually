package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/postgres/internal"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

// AggregateRepository implements the aggregate.Repository interface
// for PostgreSQL databases.
//
// This implementation uses the "aggregates" table in the database
// as its main operational table. At the same time, it also writes
// to both "events" and "event_streams" to append the Domain events
// recorded by Aggregate Roots. These updates are performed within the same transaction.
type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	conn           *pgxpool.Pool
	aggregateType  aggregate.Type[ID, T]
	aggregateSerde serde.Bytes[T]
	messageSerde   serde.Bytes[message.Message]
}

// NewAggregateRepository returns a new AggregateRepository instance.
func NewAggregateRepository[ID aggregate.ID, T aggregate.Root[ID]](
	conn *pgxpool.Pool,
	aggregateType aggregate.Type[ID, T],
	aggregateSerde serde.Bytes[T],
	messageSerde serde.Bytes[message.Message],
) AggregateRepository[ID, T] {
	return AggregateRepository[ID, T]{
		conn:           conn,
		aggregateType:  aggregateType,
		aggregateSerde: aggregateSerde,
		messageSerde:   messageSerde,
	}
}

// Get returns the aggregate.Root instance specified by the provided id.
// Returns aggregate.ErrRootNotFound if the Aggregate Root doesn't exist.
func (repo AggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	return repo.get(ctx, repo.conn, id)
}

type queryRower interface {
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

func (repo AggregateRepository[ID, T]) get(ctx context.Context, tx queryRower, id ID) (T, error) {
	var zeroValue T

	row := tx.QueryRow(
		ctx,
		`SELECT version, state
		FROM aggregates
		WHERE aggregate_id = $1 AND "type" = $2`,
		id.String(), repo.aggregateType.Name,
	)

	var (
		v     version.Version
		state []byte
	)

	if err := row.Scan(&v, &state); errors.Is(err, pgx.ErrNoRows) {
		return zeroValue, aggregate.ErrRootNotFound
	} else if err != nil {
		return zeroValue, fmt.Errorf(
			"postgres.AggregateRepository: failed to fetch aggregate state from database, %w",
			err,
		)
	}

	root, err := aggregate.RehydrateFromState(v, state, repo.aggregateSerde)
	if err != nil {
		return zeroValue, fmt.Errorf(
			"postgres.AggregateRepository: failed to deserialize state into aggregate root object, %w",
			err,
		)
	}

	return root, nil
}

// Save saves the new state of the provided aggregate.Root instance.
func (repo AggregateRepository[ID, T]) Save(ctx context.Context, root T) (err error) {
	txOpts := pgx.TxOptions{
		IsoLevel:       pgx.Serializable,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
		BeginQuery:     "",
	}

	return internal.RunTransaction(ctx, repo.conn, txOpts, func(ctx context.Context, tx pgx.Tx) error {
		eventsToCommit := root.FlushRecordedEvents()
		expectedRootVersion := root.Version() - version.Version(len(eventsToCommit))
		eventStreamID := event.StreamID(root.AggregateID().String())

		newEventStreamVersion, err := appendDomainEvents(
			ctx, tx,
			repo.messageSerde,
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

		return repo.saveAggregateState(ctx, tx, eventStreamID, root)
	})
}

func (repo AggregateRepository[ID, T]) saveAggregateState(
	ctx context.Context,
	tx pgx.Tx,
	id event.StreamID,
	root T,
) error {
	state, err := repo.aggregateSerde.Serialize(root)
	if err != nil {
		return repo.saveErr("failed to serialize aggregate root into wire format, %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO aggregates (aggregate_id, "type", "version", "state")
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (aggregate_id) DO
		UPDATE SET "version" = $3, "state" = $4`,
		id, repo.aggregateType.Name, root.Version(), state,
	); err != nil {
		return repo.saveErr("failed to save new aggregate state, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) saveErr(msg string, args ...any) error {
	return fmt.Errorf("postgres.AggregateRepository: "+msg, args...)
}

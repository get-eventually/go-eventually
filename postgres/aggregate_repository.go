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

const (
	// DefaultAggregateTableName is the default aggregate snapshot table.
	DefaultAggregateTableName = "aggregates"
	// DefaultEventsTableName is the default domain events table.
	DefaultEventsTableName = "events"
	// DefaultStreamsTableName is the default event streams table.
	DefaultStreamsTableName = "event_streams"
)

// AggregateRepository implements aggregate.Repository for PostgreSQL.
// It reads aggregate snapshots from the pool and saves snapshots and recorded
// domain events together in a self-managed Serializable transaction.
// Table names can be configured using the available functional options.
type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	conn       *pgxpool.Pool
	repository aggregateRepository[ID, T]
}

// NewAggregateRepository returns a new AggregateRepository instance.
func NewAggregateRepository[ID aggregate.ID, T aggregate.Root[ID]](
	conn *pgxpool.Pool,
	aggregateType aggregate.Type[ID, T],
	aggregateSerde serde.Bytes[T],
	messageSerde serde.Bytes[message.Message],
	options ...Option[ID, T],
) AggregateRepository[ID, T] {
	return AggregateRepository[ID, T]{
		conn:       conn,
		repository: newAggregateRepository(aggregateType, aggregateSerde, messageSerde, options...),
	}
}

// Get returns the aggregate.Root instance specified by id.
// Returns aggregate.ErrRootNotFound if the aggregate does not exist.
func (repo AggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	return repo.repository.get(ctx, repo.conn, id)
}

// Save saves the snapshot and recorded events of root in its own transaction.
func (repo AggregateRepository[ID, T]) Save(ctx context.Context, root T) error {
	txOpts := pgx.TxOptions{ //nolint:exhaustruct // We don't need all fields.
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	}

	return internal.RunTransaction(ctx, repo.conn, txOpts, func(ctx context.Context, tx pgx.Tx) error {
		return repo.repository.save(ctx, tx, root)
	})
}

// TxRetriever retrieves a caller-owned transaction from context.
type TxRetriever func(context.Context) (pgx.Tx, bool)

// ErrTransactionRequired reports a missing transaction in context.
var ErrTransactionRequired = errors.New("postgres.TransactionAwareAggregateRepository: transaction required but not found in context")

// TransactionAwareAggregateRepository implements aggregate.Repository using
// caller-owned transactions for both reads and writes. It never begins, commits,
// or rolls back a transaction. Transactions must use Serializable isolation.
// Save success means writes are staged; the caller must commit the transaction.
// After a failed or rolled-back save, discard the aggregate and reload before retrying.
type TransactionAwareAggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	retrieveTx TxRetriever
	repository aggregateRepository[ID, T]
}

// NewTransactionAwareAggregateRepository returns a repository requiring a
// transaction from retrieveTx for every operation.
func NewTransactionAwareAggregateRepository[ID aggregate.ID, T aggregate.Root[ID]](
	retrieveTx TxRetriever,
	aggregateType aggregate.Type[ID, T],
	aggregateSerde serde.Bytes[T],
	messageSerde serde.Bytes[message.Message],
	options ...Option[ID, T],
) TransactionAwareAggregateRepository[ID, T] {
	return TransactionAwareAggregateRepository[ID, T]{
		retrieveTx: retrieveTx,
		repository: newAggregateRepository(aggregateType, aggregateSerde, messageSerde, options...),
	}
}

// Get reads an aggregate snapshot within the transaction retrieved from ctx.
// Returns ErrTransactionRequired if no transaction is available, or
// aggregate.ErrRootNotFound if the aggregate does not exist.
func (repo TransactionAwareAggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	tx, ok := repo.retrieveTx(ctx)
	if !ok || tx == nil {
		var zero T

		return zero, ErrTransactionRequired
	}

	return repo.repository.get(ctx, tx, id)
}

// Save writes the snapshot and recorded events within the transaction retrieved
// from ctx. Returns ErrTransactionRequired without flushing events if no
// transaction is available. The caller must roll back on error.
func (repo TransactionAwareAggregateRepository[ID, T]) Save(ctx context.Context, root T) error {
	tx, ok := repo.retrieveTx(ctx)
	if !ok || tx == nil {
		return ErrTransactionRequired
	}

	return repo.repository.save(ctx, tx, root)
}

// Option configures the shared implementation of PostgreSQL aggregate repositories.
type Option[ID aggregate.ID, T aggregate.Root[ID]] interface {
	apply(*aggregateRepository[ID, T])
}

type option[T any] func(T)

func (apply option[T]) apply(val T) { apply(val) } //nolint:unused // Called through the generic Option interface.

// WithAggregateTableName configures the aggregate snapshot table.
func WithAggregateTableName[ID aggregate.ID, T aggregate.Root[ID]](tableName string) Option[ID, T] {
	return option[*aggregateRepository[ID, T]](func(repo *aggregateRepository[ID, T]) {
		repo.aggregateTableName = tableName
	})
}

// WithEventsTableName configures the domain events table.
func WithEventsTableName[ID aggregate.ID, T aggregate.Root[ID]](tableName string) Option[ID, T] {
	return option[*aggregateRepository[ID, T]](func(repo *aggregateRepository[ID, T]) {
		repo.eventsTableName = tableName
	})
}

// WithStreamsTableName configures the event streams table.
func WithStreamsTableName[ID aggregate.ID, T aggregate.Root[ID]](tableName string) Option[ID, T] {
	return option[*aggregateRepository[ID, T]](func(repo *aggregateRepository[ID, T]) {
		repo.streamsTableName = tableName
	})
}

type aggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	aggregateType  aggregate.Type[ID, T]
	aggregateSerde serde.Bytes[T]
	messageSerde   serde.Bytes[message.Message]

	aggregateTableName string
	eventsTableName    string
	streamsTableName   string
}

func newAggregateRepository[ID aggregate.ID, T aggregate.Root[ID]](
	aggregateType aggregate.Type[ID, T],
	aggregateSerde serde.Bytes[T],
	messageSerde serde.Bytes[message.Message],
	options ...Option[ID, T],
) aggregateRepository[ID, T] {
	repo := aggregateRepository[ID, T]{
		aggregateType:      aggregateType,
		aggregateSerde:     aggregateSerde,
		messageSerde:       messageSerde,
		aggregateTableName: DefaultAggregateTableName,
		eventsTableName:    DefaultEventsTableName,
		streamsTableName:   DefaultStreamsTableName,
	}

	for _, opt := range options {
		opt.apply(&repo)
	}

	return repo
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const getAggregateQueryTemplate = `
	SELECT version, state
	FROM %s
	WHERE aggregate_id = $1 AND "type" = $2
`

func (repo aggregateRepository[ID, T]) get(ctx context.Context, tx queryRower, id ID) (T, error) {
	var zeroValue T

	row := tx.QueryRow(
		ctx,
		fmt.Sprintf(getAggregateQueryTemplate, repo.aggregateTableName),
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

func (repo aggregateRepository[ID, T]) save(ctx context.Context, tx pgx.Tx, root T) error {
	eventsToCommit := root.FlushRecordedEvents()
	expectedRootVersion := root.Version() - version.Version(len(eventsToCommit)) //nolint:gosec // This should not overflow.
	eventStreamID := event.StreamID(root.AggregateID().String())

	newEventStreamVersion, err := appendDomainEvents(
		ctx, tx,
		repo.eventsTableName, repo.streamsTableName,
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
}

const saveAggregateQueryTemplate = `
	INSERT INTO %s (aggregate_id, "type", "version", "state")
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (aggregate_id) DO
	UPDATE SET "version" = $3, "state" = $4
`

func (repo aggregateRepository[ID, T]) saveAggregateState(
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
		fmt.Sprintf(saveAggregateQueryTemplate, repo.aggregateTableName),
		id, repo.aggregateType.Name, root.Version(), state,
	); err != nil {
		return repo.saveErr("failed to save new aggregate state, %w", err)
	}

	return nil
}

func (repo aggregateRepository[ID, T]) saveErr(msg string, args ...any) error {
	return fmt.Errorf("postgres.AggregateRepository: "+msg, args...)
}

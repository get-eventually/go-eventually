package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	// Postgres driver for migrate.
	_ "github.com/golang-migrate/migrate/database/postgres"
	"github.com/lib/pq"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

var (
	_ eventstore.Store                = &EventStore{}
	_ eventstore.SequenceNumberGetter = &EventStore{}
)

// AppendToStoreFunc represents a function type for persisting an instance of eventually.Event in postgres.EventStore.
type AppendToStoreFunc func(
	ctx context.Context,
	tx *sql.Tx,
	id stream.ID,
	expected eventstore.VersionCheck,
	event eventually.Event,
) (int64, error)

// EventStore is an eventstore.Store implementation which uses
// PostgreSQL as backend datastore.
type EventStore struct {
	db            *sql.DB
	registry      eventstore.Registry
	appendToStore AppendToStoreFunc
}

// Option defines a type for providing additional constructor adjustments for postgres.EventStore.
type Option func(EventStore) EventStore

// WithAppendMiddleware allows overriding the internal logic for appending events within a transaction.
func WithAppendMiddleware(wrap func(AppendToStoreFunc) AppendToStoreFunc) Option {
	return func(store EventStore) EventStore {
		store.appendToStore = wrap(store.appendToStore)
		return store
	}
}

// NewEventStore creates a new EventStore using the database connection pool provided.
func NewEventStore(db *sql.DB, options ...Option) EventStore {
	store := EventStore{
		db:            db,
		registry:      eventstore.NewRegistry(json.Unmarshal),
		appendToStore: appendEvent,
	}

	for _, option := range options {
		store = option(store)
	}

	return store
}

// Register registers Domain Events used by the application in order to decode events
// stored in the database by their name returned by the eventually.Payload trait.
func (st EventStore) Register(events ...eventually.Payload) error {
	return st.registry.Register(events...)
}

// Stream opens one or more Event Streams depending on the provided target.
//
// In case of multi-Event Streams targets, the Select value specified will be applied
// over the value of the Global Sequence Number of the events. In case of a single Event Stream,
// this is applied over the Version value.
func (st EventStore) Stream(
	ctx context.Context,
	es eventstore.EventStream,
	target stream.Target,
	selectt eventstore.Select,
) error {
	defer close(es)

	var (
		query string
		args  []interface{}
	)

	switch t := target.(type) {
	case stream.All:
		args = append(args, selectt.From)
		query = `SELECT * FROM events
		         WHERE global_sequence_number >= $1
		         ORDER BY global_sequence_number ASC`

	case stream.ByType:
		args = append(args, selectt.From, string(t))
		query = `SELECT * FROM events
		         WHERE global_sequence_number >= $1 AND stream_type = $2
		         ORDER BY global_sequence_number ASC`

	case stream.ByTypes:
		args = append(args, selectt.From, pq.Array(t))
		query = `SELECT * FROM events
		         WHERE global_sequence_number >= $1 AND stream_type = ANY($2)
		         ORDER BY global_sequence_number ASC`

	case stream.ByID:
		args = append(args, selectt.From, t.Type, t.Name)
		query = `SELECT * FROM events
				 WHERE "version" >= $1 AND stream_type = $2 AND stream_id = $3
				 ORDER BY "version" ASC`

	default:
		return fmt.Errorf("postgres.EventStore: unsupported stream target: %T", t)
	}

	rows, err := st.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	// FIXME(ar3s3ru): add logger support in the event store
	return rowsToStream(rows, es, st.registry, nil)
}

// Append inserts the specified Domain Events into the Event Stream specified
// by the current instance, returning the new version of the Event Stream.
//
// A version can be specified to enable an Optimistic Concurrency check
// on append, by using the expected version of the Event Stream prior
// to appending the new Events.
//
// Alternatively, VersionCheckAny can be used if no Optimistic Concurrency check
// should be carried out.
//
// NOTE: this implementation is not returning yet eventstore.ErrConflict in case
// of conflicting expectations with the provided VersionCheck value.
func (st EventStore) Append(
	ctx context.Context,
	id stream.ID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (v int64, err error) {
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to open a transaction to append: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = rollbackErr
			}
		}
	}()

	for _, event := range events {
		if v, err = st.appendToStore(ctx, tx, id, expected, event); err != nil {
			return 0, err
		}

		// Update the expected version for the next event with the new version.
		expected = eventstore.VersionCheck(v)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to commit append transaction: %w", err)
	}

	return v, nil
}

// TODO(ar3s3ru): add the ErrConflict error in case of optimistic concurrency issues.
func appendEvent(
	ctx context.Context,
	tx *sql.Tx,
	id stream.ID,
	expected eventstore.VersionCheck,
	event eventually.Event,
) (int64, error) {
	eventPayload, err := json.Marshal(event.Payload)
	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to unmarshal event payload to json: %w", err)
	}

	// To avoid null or JSONB issues.
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to unmarshal metadata to json: %w", err)
	}

	var newVersion int64

	err = tx.QueryRowContext(
		ctx,
		"SELECT append_to_store($1, $2, $3, $4, $5, $6)",
		id.Type,
		id.Name,
		int64(expected),
		event.Payload.Name(),
		eventPayload,
		metadata,
	).Scan(&newVersion)

	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to append event: %w", err)
	}

	return newVersion, nil
}

// LatestSequenceNumber returns the latest Sequence Number used by a Domain Event
// committed to the Event Store.
func (st EventStore) LatestSequenceNumber(ctx context.Context) (int64, error) {
	row := st.db.QueryRowContext(
		ctx,
		"SELECT max(global_sequence_number) FROM events",
	)

	if err := row.Err(); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to get latest sequence number: %w", err)
	}

	var sequenceNumber int64
	if err := row.Scan(&sequenceNumber); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to scan latest sequence number from sql row: %w", err)
	}

	return sequenceNumber, nil
}

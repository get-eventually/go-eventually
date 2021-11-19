package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

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

// AppendToStoreFunc is the function type used by the postgres.EventStore
// to delegate the append call to the database instace.
type AppendToStoreFunc func(
	ctx context.Context,
	tx *sql.Tx,
	id stream.ID,
	expected eventstore.VersionCheck,
	eventName string,
	payload []byte,
	metadata []byte,
) (int64, error)

// EventStore is an eventstore.Store implementation which uses
// PostgreSQL as backend datastore.
type EventStore struct {
	db            *sql.DB
	serde         Serde
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
func NewEventStore(db *sql.DB, serde Serde, options ...Option) EventStore {
	store := EventStore{
		db:    db,
		serde: serde,
	}

	store.appendToStore = performAppendQuery

	for _, option := range options {
		store = option(store)
	}

	return store
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
	return rowsToStream(rows, es, st.serde, nil)
}

//nolint:lll // It's ok to go over the 120 lines limit in this case.
var conflictErrorRegex = regexp.MustCompile(`stream version check failed, expected: (?P<expected>\d), current: (?P<actual>\d)`)

func handleAppendError(err error) error {
	var dbErr *pq.Error
	if !errors.As(err, &dbErr) {
		return err
	}

	matches := conflictErrorRegex.FindStringSubmatch(dbErr.Message)
	if len(matches) == 0 {
		return err
	}

	expected, err := strconv.Atoi(matches[conflictErrorRegex.SubexpIndex("expected")])
	if err != nil {
		return fmt.Errorf("postgres.EventStore.handleAppendError: failed to parse conflict error expected version: %w", err)
	}

	actual, err := strconv.Atoi(matches[conflictErrorRegex.SubexpIndex("actual")])
	if err != nil {
		return fmt.Errorf("postgres.EventStore.handleAppendError: failed to parse conflict error actual version: %w", err)
	}

	return eventstore.ConflictError{
		Expected: int64(expected),
		Actual:   int64(actual),
	}
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
		if v, err = st.appendEvent(ctx, tx, id, expected, event); err != nil {
			return 0, fmt.Errorf("postgres.EventStore: failed to append event: %w", handleAppendError(err))
		}

		// Update the expected version for the next event with the new version.
		expected = eventstore.VersionCheck(v)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to commit append transaction: %w", err)
	}

	return v, nil
}

func performAppendQuery(
	ctx context.Context,
	tx *sql.Tx,
	id stream.ID,
	expected eventstore.VersionCheck,
	eventName string,
	payload []byte,
	metadata []byte,
) (int64, error) {
	var newVersion int64

	if err := tx.QueryRowContext(
		ctx,
		"SELECT append_to_store($1, $2, $3, $4, $5, $6)",
		id.Type,
		id.Name,
		int64(expected),
		eventName,
		payload,
		metadata,
	).Scan(&newVersion); err != nil {
		return 0, fmt.Errorf("postgres.EventStore.performAppendQuery: failed to append event: %w", err)
	}

	return newVersion, nil
}

func (st EventStore) appendEvent(
	ctx context.Context,
	tx *sql.Tx,
	id stream.ID,
	expected eventstore.VersionCheck,
	event eventually.Event,
) (int64, error) {
	eventPayload, err := st.serde.Serialize(event.Payload.Name(), id, event.Payload)
	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to serialize event payload: %w", err)
	}

	// To avoid null or JSONB issues.
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to marshal metadata to json: %w", err)
	}

	return st.appendToStore(ctx, tx, id, expected, event.Payload.Name(), eventPayload, metadata)
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

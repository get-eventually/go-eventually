package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"

	_ "github.com/golang-migrate/migrate/database/postgres" // postgres driver for migrate
)

var _ eventstore.Store = &EventStore{}

// EventStore is an eventstore.Store implementation which uses
// PostgreSQL as backend datastore.
type EventStore struct {
	db       *sql.DB
	registry eventstore.Registry
}

func NewEventStore(db *sql.DB) EventStore {
	return EventStore{
		db:       db,
		registry: eventstore.NewRegistry(json.Unmarshal),
	}
}

// Register registers Domain Events used by the application in order to decode events
// stored in the database by their name returned by the eventually.Payload trait.
func (st EventStore) Register(events ...eventually.Payload) error {
	return st.registry.Register(events...)
}

// StreamAll opens an Event Stream and sinks all the events in the Event Store in the provided
// channel, skipping those events with a sequence number lower than the provided bound.
func (st EventStore) StreamAll(ctx context.Context, es eventstore.EventStream, selectt eventstore.Select) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE global_sequence_number >= $1
			ORDER BY global_sequence_number ASC`,
		selectt.From,
	)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	// FIXME(ar3s3ru): add logger support in the event store
	return rowsToStream(rows, es, st.registry, nil)
}

// StreamByType opens a stream of all Event Streams grouped by the same Type,
// as specified in input.
//
// The stream will be ordered based on their Global Sequence Number.
func (st EventStore) StreamByType(
	ctx context.Context,
	es eventstore.EventStream,
	typ string,
	selectt eventstore.Select,
) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE stream_type = $1 AND global_sequence_number >= $2
			ORDER BY global_sequence_number ASC`,
		typ,
		selectt.From,
	)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	// FIXME(ar3s3ru): add logger support in the event store
	return rowsToStream(rows, es, st.registry, nil)
}

// Stream opens the specific Event Stream identified by the provided id.
func (st EventStore) Stream(
	ctx context.Context,
	es eventstore.EventStream,
	id eventstore.StreamID,
	selectt eventstore.Select,
) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE stream_type = $1 AND stream_id = $2 AND "version" >= $3
			ORDER BY "version" ASC`,
		id.Type,
		id.Name,
		selectt.From,
	)
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
	id eventstore.StreamID,
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
func (st *EventStore) appendEvent(
	ctx context.Context,
	tx *sql.Tx,
	id eventstore.StreamID,
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

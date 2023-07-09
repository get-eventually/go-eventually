package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

var _ event.Store = EventStore{}

// EventStore is an event.Store implementation targeted to PostgreSQL databases.
//
// The implementation uses "event_streams" and "events" as their
// operational tables. Updates to these tables are transactional.
type EventStore struct {
	Conn  *pgxpool.Pool
	Serde serde.Bytes[message.Message]
}

// Stream implements the event.Streamer interface.
func (es EventStore) Stream(
	ctx context.Context,
	stream event.StreamWrite,
	id event.StreamID,
	selector version.Selector,
) error {
	defer close(stream)

	rows, err := es.Conn.Query(
		ctx,
		`SELECT version, event, metadata FROM events
		WHERE event_stream_id = $1 AND version >= $2
		ORDER BY version`,
		id, selector.From,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to query events table: %w", err)
	}

	for rows.Next() {
		var (
			rawEvent    []byte
			rawMetadata json.RawMessage
		)

		evt := event.Persisted{
			StreamID: id,
		}

		if err := rows.Scan(&evt.Version, &rawEvent, &rawMetadata); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to scan next row")
		}

		msg, err := es.Serde.Deserialize(rawEvent)
		if err != nil {
			return fmt.Errorf("postgres.EventStore: failed to deserialize event: %w", err)
		}

		evt.Message = msg

		if err := json.Unmarshal(rawMetadata, &evt.Metadata); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to deserialize metadata: %w", err)
		}

		stream <- evt
	}

	return nil
}

// Append implements event.Store.
func (es EventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	tx, err := es.Conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	})
	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to open database transaction: %w", err)
	}

	defer func() {
		// NOTE: should not have effect if the transaction has been committed
		_ = tx.Rollback(ctx)
	}()

	var newVersion version.Version

	switch v := expected.(type) {
	case version.CheckExact:
		expectedVersion := version.Version(v)
		newVersion = expectedVersion + version.Version(len(events))

		_, err := tx.Exec(
			ctx,
			"CALL upsert_event_stream($1::TEXT, $2::INTEGER, $3::INTEGER)",
			id, expectedVersion, newVersion,
		)

		if conflictErr, ok := isVersionConflictError(err); ok {
			return 0, conflictErr
		}

		if err != nil {
			return 0, fmt.Errorf("postgres.EventStore: failed to update event stream version, %w", err)
		}

	case version.CheckAny:
		row := tx.QueryRow(
			ctx,
			"SELECT * FROM upsert_event_stream_with_no_version_check($1::TEXT, $2::INTEGER)",
			id, len(events),
		)

		if err := row.Scan(&newVersion); err != nil {
			return 0, fmt.Errorf("postgres.EventStore: failed to update event stream version unchecked: %w", err)
		}

	default:
		return 0, fmt.Errorf("postgres.EventStore: unexpected 'expected' version check type, %T", v)
	}

	if err := appendDomainEvents(ctx, tx, es.Serde, id, newVersion, events...); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to append domain events: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to commit transaction, %w", err)
	}

	return newVersion, nil
}

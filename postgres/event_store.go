package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/postgres/internal"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

//nolint:exhaustruct // Interface implementation assertion.
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
		return fmt.Errorf("postgres.EventStore: failed to query events table, %w", err)
	}

	for rows.Next() {
		var (
			rawEvent     []byte
			rawMetadata  json.RawMessage
			eventVersion version.Version
		)

		if err := rows.Scan(&eventVersion, &rawEvent, &rawMetadata); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to scan next row")
		}

		msg, err := es.Serde.Deserialize(rawEvent)
		if err != nil {
			return fmt.Errorf("postgres.EventStore: failed to deserialize event, %w", err)
		}

		var metadata message.Metadata
		if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to deserialize metadata, %w", err)
		}

		stream <- event.Persisted{
			StreamID: id,
			Version:  eventVersion,
			Envelope: event.Envelope{
				Message:  msg,
				Metadata: metadata,
			},
		}
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
	var newVersion version.Version

	txOpts := pgx.TxOptions{
		IsoLevel:       pgx.Serializable,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
		BeginQuery:     "",
	}

	if err := internal.RunTransaction(ctx, es.Conn, txOpts, func(ctx context.Context, tx pgx.Tx) error {
		var err error

		if newVersion, err = appendDomainEvents(ctx, tx, es.Serde, id, expected, events...); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to append domain events, %w", err)
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return newVersion, nil
}

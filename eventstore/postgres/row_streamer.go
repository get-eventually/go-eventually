package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/logger"
)

func rowsToStream(rows *sql.Rows, es eventstore.EventStream, deserializer Deserializer, l logger.Logger) error {
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error(l, "Failed to close streamed event rows", logger.With("err", err))
		}
	}()

	for rows.Next() {
		var (
			eventName               string
			event                   eventstore.Event
			rawPayload, rawMetadata json.RawMessage
		)

		err := rows.Scan(
			&event.SequenceNumber,
			&event.Stream.Type,
			&event.Stream.Name,
			&eventName,
			&event.Version,
			&rawPayload,
			&rawMetadata,
		)
		if err != nil {
			return fmt.Errorf("postgres.EventStore: failed to scan stream row into event struct: %w", err)
		}

		payload, err := deserializer.Deserialize(eventName, event.Stream, rawPayload)
		if err != nil {
			return fmt.Errorf("postgres.EventStore: failed to deserialize event: %w", err)
		}

		var metadata eventually.Metadata
		if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to unmarshal event metadata from json: %w", err)
		}

		event.Payload = payload
		event.Metadata = metadata

		es <- event
	}

	return nil
}

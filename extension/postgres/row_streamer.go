package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/event"
)

func rowsToStream(rows *sql.Rows, es event.Stream, deserializer Deserializer, l eventually.Logger) error {
	defer func() {
		if err := rows.Close(); err != nil {
			l.LogErrorf(func(log eventually.LoggerFunc) {
				log("Failed to close streamed event rows: %s", err)
			})
		}
	}()

	for rows.Next() {
		var (
			eventName               string
			evt                     event.Persisted
			rawPayload, rawMetadata json.RawMessage
		)

		if err := rows.Scan(
			&evt.Stream.Type,
			&evt.Stream.Name,
			&eventName,
			&evt.Version,
			&rawPayload,
			&rawMetadata,
		); err != nil {
			return fmt.Errorf("postgres.EventStore.rowsToStream: failed to scan stream row into event struct: %w", err)
		}

		payload, err := deserializer.Deserialize(eventName, rawPayload)
		if err != nil {
			return fmt.Errorf("postgres.EventStore.rowsToStream: failed to deserialize event: %w", err)
		}

		var metadata eventually.Metadata
		if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
			return fmt.Errorf("postgres.EventStore.rowsToStream: failed to unmarshal event metadata from json: %w", err)
		}

		evt.Payload = payload
		evt.Metadata = metadata

		es <- evt
	}

	return nil
}

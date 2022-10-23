package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

func serializeMetadata(metadata message.Metadata) ([]byte, error) {
	if metadata == nil {
		return nil, nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("postgres.serializeMetadata: failed to marshal to json, %w", err)
	}

	return data, nil
}

func appendDomainEvent(
	ctx context.Context,
	tx pgx.Tx,
	messageSerializer serde.Serializer[message.Message, []byte],
	id event.StreamID,
	eventVersion, newVersion version.Version,
	evt event.Envelope,
) error {
	msg := evt.Message

	data, err := messageSerializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("postgres.appendDomainEvent: failed to serialize domain event, %w", err)
	}

	enrichedMetadata := evt.Metadata.
		With("Recorded-At", time.Now().Format(time.RFC3339Nano)).
		With("Recorded-With-New-Overall-Version", strconv.Itoa(int(newVersion)))

	metadata, err := serializeMetadata(enrichedMetadata)
	if err != nil {
		return err
	}

	if _, err = tx.Exec(
		ctx,
		`INSERT INTO events (event_stream_id, "type", "version", event, metadata) VALUES ($1, $2, $3, $4, $5)`,
		id, msg.Name(), eventVersion, data, metadata,
	); err != nil {
		return fmt.Errorf("postgres.appendDomainEvent: failed to append new domain event to event store, %w", err)
	}

	return nil
}

func appendDomainEvents(
	ctx context.Context,
	tx pgx.Tx,
	messageSerializer serde.Serializer[message.Message, []byte],
	id event.StreamID,
	newVersion version.Version,
	events ...event.Envelope,
) error {
	currentVersion := newVersion - version.Version(len(events))

	for i, event := range events {
		eventVersion := currentVersion + version.Version(i) + 1

		err := appendDomainEvent(ctx, tx, messageSerializer, id, eventVersion, newVersion, event)
		if err != nil {
			return err
		}
	}

	return nil
}

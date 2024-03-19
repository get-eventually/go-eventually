package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

func appendDomainEvents(
	ctx context.Context,
	tx pgx.Tx,
	messageSerializer serde.Serializer[message.Message, []byte],
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	row := tx.QueryRow(
		ctx,
		`SELECT version FROM event_streams WHERE event_stream_id = $1`,
		id,
	)

	var oldVersion version.Version
	if err := row.Scan(&oldVersion); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("postgres.appendDomainEvents: failed to scan old event stream version, %w", err)
	}

	if v, ok := expected.(version.CheckExact); ok && oldVersion != version.Version(v) {
		return 0, fmt.Errorf(
			"eventuallypostges.appendDomainEvents: event stream version check failed, %w",
			version.ConflictError{
				Expected: version.Version(v),
				Actual:   oldVersion,
			},
		)
	}

	newVersion := oldVersion + version.Version(len(events))

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO event_streams (event_stream_id, version)
		VALUES ($1, $2)
		ON CONFLICT (event_stream_id) DO
		UPDATE SET version = $2`,
		id, newVersion,
	); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to update event stream, %w", err)
	}

	for i, event := range events {
		eventVersion := oldVersion + version.Version(i) + 1

		if err := appendDomainEvent(ctx, tx, messageSerializer, id, eventVersion, newVersion, event); err != nil {
			return 0, err
		}
	}

	return newVersion, nil
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
		`INSERT INTO events (event_stream_id, "type", "version", event, metadata)
		VALUES ($1, $2, $3, $4, $5)`,
		id, msg.Name(), eventVersion, data, metadata,
	); err != nil {
		return fmt.Errorf("postgres.appendDomainEvent: failed to append new domain event to event store, %w", err)
	}

	return nil
}

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

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/version"
)

type AggregateSerializer[ID aggregate.ID, T aggregate.Root[ID]] interface {
	aggregate.Serializer[ID, T, []byte]
}

type AggregateDeserializer[ID aggregate.ID, T aggregate.Root[ID]] interface {
	aggregate.Deserializer[ID, []byte, T]
}

type AggregateSerde[ID aggregate.ID, T aggregate.Root[ID]] interface {
	AggregateSerializer[ID, T]
	AggregateDeserializer[ID, T]
}

type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	Conn              *pgx.Conn
	AggregateTypeName string
	AggregateSerde    AggregateSerde[ID, T]
	MessageSerde      MessageSerde
}

func (repo AggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	var zeroValue T

	row := repo.Conn.QueryRow(
		ctx,
		`SELECT version, state
		FROM aggregates
		WHERE aggregate_id = $1 AND "type" = $2`,
		id.String(), repo.AggregateTypeName,
	)

	var (
		v     version.Version
		state []byte
	)

	err := row.Scan(&v, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return zeroValue, aggregate.ErrRootNotFound
	}
	if err != nil {
		return zeroValue, fmt.Errorf(
			"postgres.AggregateRepository.Get: failed to fetch aggregate state from database: %w",
			err,
		)
	}

	root, err := aggregate.RehydrateFromState[ID, []byte, T](v, state, repo.AggregateSerde)
	if err != nil {
		return zeroValue, fmt.Errorf(
			"postgres.AggregateRepository.Get: failed to deserialize state into aggregate root object: %w",
			err,
		)
	}

	return root, nil
}

func (repo AggregateRepository[ID, T]) saveErr(msg string, args ...any) error {
	return fmt.Errorf("postgres.AggregateRepository.Save: "+msg, args...)
}

func (repo AggregateRepository[ID, T]) saveAggregateState(
	ctx context.Context,
	tx pgx.Tx,
	expectedVersion version.Version,
	root T,
) error {
	state, err := repo.AggregateSerde.Serialize(root)
	if err != nil {
		return repo.saveErr("failed to serialize aggregate root into wire format, %w", err)
	}

	// TODO(ar3s3ru): need to add version check... maybe using a procedure?

	if _, err = tx.Exec(
		ctx,
		`INSERT INTO aggregates (aggregate_id, "type", "version", "state") VALUES ($1, $2, $3, $4)
		ON CONFLICT (aggregate_id, "type") DO
		UPDATE SET "version" = $3, "state" = $4`,
		root.AggregateID().String(), repo.AggregateTypeName, root.Version(), state,
	); err != nil {
		return repo.saveErr("failed to save new aggregate state, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) deserializeMetadata(metadata message.Metadata) ([]byte, error) {
	if metadata == nil {
		return nil, nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, repo.saveErr("failed to serialize metadata to json, %w", err)
	}

	return data, nil
}

func (repo AggregateRepository[ID, T]) appendDomainEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventStreamID event.StreamID,
	eventVersion version.Version,
	event event.Envelope,
) error {
	msg := event.Message

	data, err := repo.MessageSerde.Serialize(msg)
	if err != nil {
		return repo.saveErr("failed to serialize domain event, %w", err)
	}

	metadata, err := repo.deserializeMetadata(event.Metadata)
	if err != nil {
		return err
	}

	if _, err = tx.Exec(
		ctx,
		`INSERT INTO events (event_stream_id, "type", "version", event, metadata) VALUES ($1, $2, $3, $4, $5)`,
		eventStreamID, msg.Name(), eventVersion, data, metadata,
	); err != nil {
		return repo.saveErr("failed to append new domain event to event store, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) appendDomainEvents(
	ctx context.Context,
	tx pgx.Tx,
	aggregateID ID,
	lastAggregateVersion version.Version,
	events ...event.Envelope,
) error {
	eventStreamID := event.StreamID(aggregateID.String())
	currentAggregateVersion := lastAggregateVersion - version.Version(len(events))

	for i, event := range events {
		eventVersion := currentAggregateVersion + version.Version(i) + 1

		if err := repo.appendDomainEvent(ctx, tx, eventStreamID, eventVersion, event); err != nil {
			return err
		}
	}

	return nil
}

func (repo AggregateRepository[ID, T]) Save(ctx context.Context, root T) error {
	conn := repo.Conn

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.NotDeferrable,
	})

	if err != nil {
		return repo.saveErr("failed to open db transaction, %w", err)
	}

	defer func() {
		// NOTE: should not have effect if the transaction has been committed
		_ = tx.Rollback(ctx)
	}()

	eventsToCommit := root.FlushRecordedEvents()
	expectedRootVersion := root.Version() - version.Version(len(eventsToCommit))

	if err := repo.saveAggregateState(ctx, tx, expectedRootVersion, root); err != nil {
		return err
	}

	if err := repo.appendDomainEvents(ctx, tx, root.AggregateID(), root.Version(), eventsToCommit...); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.saveErr("failed to commit transaction, %w", err)
	}

	return nil
}

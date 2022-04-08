package postgresql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"

	"github.com/jackc/pgx/v4"
)

type AggregateSerializer[ID aggregate.ID, T aggregate.Root[ID]] interface {
	aggregate.Serializer[ID, T, []byte]
}

type AggregateDeserializer[ID aggregate.ID, T aggregate.Root[ID]] interface {
	aggregate.Deserializer[ID, T, []byte]
}

type AggregateSerde[ID aggregate.ID, T aggregate.Root[ID]] interface {
	AggregateSerializer[ID, T]
	AggregateDeserializer[ID, T]
}

type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	Conn           *pgx.Conn
	Table          string
	AggregateSerde AggregateSerde[ID, T]
	MessageSerde   MessageSerde
}

func (repo AggregateRepository[ID, T]) Get(ctx context.Context, id ID) (T, error) {
	var zeroValue T

	conn := repo.Conn
	row := conn.QueryRow(ctx, "SELECT version, state FROM $1 WHERE aggregate_id = $2", repo.Table, id.String())

	var (
		v     version.Version
		state []byte
	)

	err := row.Scan(&v, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return zeroValue, aggregate.ErrRootNotFound
	}
	if err != nil {
		return zeroValue, fmt.Errorf("%T.Get: failed to fetch aggregate state from database: %w", repo, err)
	}

	root, err := repo.AggregateSerde.Deserialize(v, state)
	if err != nil {
		return zeroValue, fmt.Errorf("%T.Get: failed to deserialize state into aggregate root object: %w", repo, err)
	}

	return root, nil
}

func (repo AggregateRepository[ID, T]) saveErr(msg string, args ...any) error {
	return fmt.Errorf("%T.Save: "+msg, args)
}

func (repo AggregateRepository[ID, T]) saveAggregateState(ctx context.Context, tx pgx.Tx, root T) error {
	state, err := repo.AggregateSerde.Serialize(root)
	if err != nil {
		return repo.saveErr("failed to serialize aggregate root into wire format, %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		`INSERT INTO $1 (aggregate_id, version, state) VALUES ($2, $3, $4)
		ON CONFLICT (aggregate_id) DO
		UPDATE SET version = $3, state = $4`,
		repo.Table, root.AggregateID().String(), root.Version(), state,
	); err != nil {
		return repo.saveErr("failed to save new aggregate state, %w", err)
	}

	return nil
}

func (repo AggregateRepository[ID, T]) appendDomainEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventStreamID event.StreamID,
	eventVersion version.Version,
	event event.Envelope,
) error {
	msg := event.Message

	data, err := repo.MessageSerde.Serialize(msg.Name(), eventStreamID, msg)
	if err != nil {
		return repo.saveErr("failed to serialize domain event, %w", err)
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return repo.saveErr("failed to serialize metadata to json, %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		"INSERT INTO events (event_stream_id, version, event, metadata) VALUES ($1, $2, $3, $4)",
		eventStreamID, eventVersion, data, metadata,
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

	if err := repo.saveAggregateState(ctx, tx, root); err != nil {
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

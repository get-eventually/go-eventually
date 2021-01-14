package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/postgres/migrations"
	"github.com/eventually-rs/eventually-go/subscription"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres" // postgres driver for migrate
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/lib/pq"
)

const streamAllName = "$all"

var _ eventstore.Store = &EventStore{}
var _ subscription.Checkpointer = &EventStore{}

type EventStore struct {
	dsn             string
	db              *sql.DB
	eventNameToType map[string]reflect.Type
	eventTypeToName map[reflect.Type]string
}

func OpenEventStore(dsn string) (*EventStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.EventStore: failed to open connection with the db: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		return nil, err
	}

	return &EventStore{
		db:              db,
		dsn:             dsn,
		eventNameToType: make(map[string]reflect.Type),
		eventTypeToName: make(map[reflect.Type]string),
	}, nil
}

func runMigrations(dsn string) (err error) {
	src := bindata.Resource(migrations.AssetNames(), func(name string) ([]byte, error) {
		return migrations.Asset(name)
	})

	driver, err := bindata.WithInstance(src)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to access migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", driver, dsn)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres.EventStore: failed to migrate database: %w", err)
	}

	return nil
}

func (st *EventStore) Close() error {
	return st.db.Close()
}

func (st *EventStore) Get(ctx context.Context, subscriptionName string) (int64, error) {
	row := st.db.QueryRowContext(
		ctx,
		"SELECT get_or_create_subscription_checkpoint($1)",
		subscriptionName,
	)

	var lastSequenceNumber int64
	if err := row.Scan(&lastSequenceNumber); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to get subscription checkpoint: %w", err)
	}

	return lastSequenceNumber, nil
}

func (st *EventStore) Store(ctx context.Context, subscriptionName string, sequenceNumber int64) error {
	_, err := st.db.ExecContext(
		ctx,
		`UPDATE subscriptions_checkpoints
		SET last_sequence_number = $1
		WHERE subscription_id = $2`,
		sequenceNumber,
		subscriptionName,
	)

	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to store subscription checkpoint: %w", err)
	}

	return nil
}

func (st *EventStore) Register(ctx context.Context, typ string, events map[string]interface{}) error {
	if err := st.registerEventsToType(events); err != nil {
		return fmt.Errorf("postgres.EventStore: failed to register types: %w", err)
	}

	return nil
}

func (st *EventStore) registerEventsToType(events map[string]interface{}) error {
	for eventName, event := range events {
		eventType := reflect.TypeOf(event)

		if _, ok := st.eventNameToType[eventName]; ok {
			return fmt.Errorf("postgres.EventStore: event '%s' already registered", eventName)
		}

		st.eventNameToType[eventName] = eventType
		st.eventTypeToName[eventType] = eventName
	}

	return nil
}

func (st *EventStore) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	// TODO: query the database to check stream_types
	return &typedEventStore{
		parent:     st,
		streamType: typ,
	}, nil
}

func (st *EventStore) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE global_sequence_number > $1
			ORDER BY global_sequence_number ASC`,
		from,
	)

	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	return st.rowsToStream(rows, es)
}

type rawNotificationEvent struct {
	StreamID   string              `json:"stream_id"`
	StreamType string              `json:"stream_type"`
	EventType  string              `json:"event_type"`
	Version    int64               `json:"version"`
	Event      json.RawMessage     `json:"event"`
	Metadata   eventually.Metadata `json:"metadata"`
}

func (st *EventStore) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	return st.subscribe(ctx, streamAllName, es)
}

func (st *EventStore) subscribe(ctx context.Context, name string, es eventstore.EventStream) error {
	defer close(es)

	listener := pq.NewListener(st.dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	})

	// TODO: proper error handling!
	defer listener.Close()

	if err := listener.Listen(name); err != nil {
		return fmt.Errorf("postgres.EventStore: failed to listen on stream: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres.EventStore: listener closed: %w", ctx.Err())

		case <-time.After(10 * time.Second):
			if err := listener.Ping(); err != nil {
				return fmt.Errorf("postgres.EventStore: failed to ping listener: %w", err)
			}

		case notification := <-listener.Notify:
			if notification == nil {
				continue
			}

			buffer := bytes.NewBufferString(notification.Extra)

			var rawEvent rawNotificationEvent
			if err := json.NewDecoder(buffer).Decode(&rawEvent); err != nil {
				return fmt.Errorf("postgres.EventStore: failed to unmarshal notification payload into event; %w", err)
			}

			payload, ok := st.eventNameToType[rawEvent.EventType]
			if !ok {
				return fmt.Errorf("postgres.EventStore: received unregistered event '%s'", rawEvent.EventType)
			}

			vp := reflect.New(payload)
			if err := json.Unmarshal(rawEvent.Event, vp.Interface()); err != nil {
				return fmt.Errorf("postgres.EventStore: failed to unmarshal event payload from json: %w", err)
			}

			es <- eventstore.Event{
				StreamType: rawEvent.StreamType,
				StreamName: rawEvent.StreamID,
				Version:    rawEvent.Version,
				Event: eventually.Event{
					Payload:  vp.Elem().Interface(),
					Metadata: rawEvent.Metadata,
				},
			}
		}
	}
}

type typedEventStore struct {
	parent     *EventStore
	streamType string
}

func (st *typedEventStore) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	defer close(es)

	db := st.parent.db

	rows, err := db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE stream_type = $1 AND global_sequence_number > $2
			ORDER BY global_sequence_number ASC`,
		st.streamType,
		from,
	)

	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	return st.parent.rowsToStream(rows, es)
}

func (st *typedEventStore) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	return st.parent.subscribe(ctx, st.streamType, es)
}

func (st *typedEventStore) Instance(id string) eventstore.Instanced {
	return &instancedEventStore{
		parent:   st,
		streamID: id,
	}
}

type instancedEventStore struct {
	parent   *typedEventStore
	streamID string
}

func (st *instancedEventStore) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	defer close(es)

	db := st.parent.parent.db
	streamType := st.parent.streamType

	rows, err := db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE stream_type = $1 AND stream_id = $2 AND "version" > $3
			ORDER BY "version" ASC`,
		streamType,
		st.streamID,
		from,
	)

	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	return st.parent.parent.rowsToStream(rows, es)
}

func (st *instancedEventStore) Append(ctx context.Context, version int64, events ...eventually.Event) (v int64, err error) {
	db := st.parent.parent.db
	streamType := st.parent.streamType

	tx, err := db.BeginTx(ctx, nil)
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
		eventType := reflect.TypeOf(event.Payload)
		eventName, ok := st.parent.parent.eventTypeToName[eventType]
		if !ok {
			return 0, fmt.Errorf("postgres.EventStore: event type not registered: %s", eventType.Name())
		}

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

		err = tx.QueryRowContext(
			ctx,
			"SELECT append_to_store($1, $2, $3, $4, $5, $6)",
			streamType,
			st.streamID,
			version,
			eventName,
			eventPayload,
			metadata,
		).Scan(&version)

		if err != nil {
			return 0, fmt.Errorf("postgres.EventStore: failed to append event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to commit append transaction: %w", err)
	}

	return version, nil
}

func (st *EventStore) rowsToStream(rows *sql.Rows, es eventstore.EventStream) (err error) {
	defer func() {
		if err != nil {
			return
		}

		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("postgres.EventStore: failed to close stream rows: %w", closeErr)
		}
	}()

	for rows.Next() {
		var (
			globalSequenceNumber    int64
			eventName               string
			event                   eventstore.Event
			rawPayload, rawMetadata json.RawMessage
		)

		err := rows.Scan(
			&globalSequenceNumber,
			&event.StreamType,
			&event.StreamName,
			&eventName,
			&event.Version,
			&rawPayload,
			&rawMetadata,
		)

		if err != nil {
			return fmt.Errorf("postgres.EventStore: failed to scan stream row into event struct: %w", err)
		}

		payload, ok := st.eventNameToType[eventName]
		if !ok {
			return fmt.Errorf("postgres.EventStore: received unregistered event '%s'", eventName)
		}

		vp := reflect.New(payload)
		if err := json.Unmarshal(rawPayload, vp.Interface()); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to unmarshal event payload from json: %w", err)
		}

		var metadata eventually.Metadata
		if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
			return fmt.Errorf("postgres.EventStore: failed to unmarshal event metadata from json: %w", err)
		}

		metadata.WithGlobalSequenceNumber(globalSequenceNumber)

		event.Payload = vp.Elem().Interface()
		event.Metadata = metadata

		es <- event
	}

	return nil
}

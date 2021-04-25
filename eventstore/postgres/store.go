package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/postgres/migrations"
	"github.com/eventually-rs/eventually-go/subscription/checkpoint"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres" // postgres driver for migrate
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/lib/pq"
)

const (
	streamAllName = "$all"

	// DefaultNotifyChannelTimeout is the default refresh timeout for each
	// notifications received through LISTEN.
	DefaultNotifyChannelTimeout = 10 * time.Second

	// DefaultReconnectionTimeout is the minimum timeout value the database driver
	// uses before re-establishing a connection with the database when
	// the previous one had been closed.
	DefaultReconnectionTimeout = 10 * time.Second
)

var (
	// ErrEmptyEventsMap occurs during a call to Register where a nil or empty Events map
	// is provided, which would mean no events would be registered for the desired type.
	ErrEmptyEventsMap = fmt.Errorf("postgres.EventStore: empty events map provided for type")
)

var (
	_ eventstore.Store        = &EventStore{}
	_ checkpoint.Checkpointer = &EventStore{}
)

// EventStore is an eventstore.Store implementation which uses
// PostgreSQL as backend datastore.
type EventStore struct {
	dsn             string
	db              *sql.DB
	eventNameToType map[string]reflect.Type
	eventTypeToName map[reflect.Type]string
}

// OpenEventStore opens a connection with the PostgreSQL identified by the
// provided DSN and run migrations for the Event Store functionalities.
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
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: invalid dsn format: %w", err)
	}

	// go-migrate allows to specify a different migration table
	// than the default 'schema_migrations'. In this case, we want to use
	// a dedicated table to avoid potential clashing with the same tool running
	// on the same PostgreSQL database instance that is being used as
	// an Event Store.
	q := u.Query()
	q.Add("x-migrations-table", "eventually_schema_migrations")
	u.RawQuery = q.Encode()

	src := bindata.Resource(migrations.AssetNames(), migrations.Asset)

	driver, err := bindata.WithInstance(src)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to access migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", driver, u.String())
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres.EventStore: failed to migrate database: %w", err)
	}

	return nil
}

// Close closes the Event Store database connection.
func (st *EventStore) Close() error {
	return st.db.Close()
}

// Read reads the latest checkpointed sequence number of the subscription specified.
func (st *EventStore) Read(ctx context.Context, subscriptionName string) (int64, error) {
	row := st.db.QueryRowContext(
		ctx,
		"SELECT get_or_create_subscription_checkpoint($1)",
		subscriptionName,
	)

	var lastSequenceNumber int64
	if err := row.Scan(&lastSequenceNumber); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to read subscription checkpoint: %w", err)
	}

	return lastSequenceNumber, nil
}

// Write checkpoints the sequence number value provided for the specified subscription.
func (st *EventStore) Write(ctx context.Context, subscriptionName string, sequenceNumber int64) error {
	_, err := st.db.ExecContext(
		ctx,
		`UPDATE subscriptions_checkpoints
		SET last_sequence_number = $1
		WHERE subscription_id = $2`,
		sequenceNumber,
		subscriptionName,
	)

	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to write subscription checkpoint: %w", err)
	}

	return nil
}

// Register adds a mapping between the Stream Type identifier and the Events Map provided,
// which is necessary to decode the Event payloads written to the database.
//
// The Event Map should use an unique identifier for the event, and a zero-valued instance
// of the Event type it corresponds to.
func (st *EventStore) Register(ctx context.Context, typ string, events ...eventually.Payload) error {
	if len(events) == 0 {
		return ErrEmptyEventsMap
	}

	if err := st.registerEventsToType(events...); err != nil {
		return fmt.Errorf("postgres.EventStore: failed to register types: %w", err)
	}

	return nil
}

func (st *EventStore) registerEventsToType(events ...eventually.Payload) error {
	for _, event := range events {
		eventName := event.Name()
		eventType := reflect.TypeOf(event)

		if _, ok := st.eventNameToType[eventName]; ok {
			return fmt.Errorf("postgres.EventStore: event '%s' already registered", eventName)
		}

		st.eventNameToType[eventName] = eventType
		st.eventTypeToName[eventType] = eventName
	}

	return nil
}

// Type returns an eventstore.Typed access instance of the specified Stream Type,
// if previously registered.
func (st *EventStore) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	// TODO: query the database to check stream_types
	return &typedEventStore{
		parent:     st,
		streamType: typ,
	}, nil
}

// Stream opens an Event Stream and sinks all the events in the Event Store in the provided
// channel, skipping those events with a sequence number lower than the provided bound.
func (st *EventStore) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE global_sequence_number >= $1
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

// Subscribe subscribes to all the new Events committed to the Event Store
// and sinks them in the provided channel.
func (st *EventStore) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	return st.subscribe(ctx, streamAllName, es)
}

func (st *EventStore) subscribe(ctx context.Context, name string, es eventstore.EventStream) error {
	defer close(es)

	listener := pq.NewListener(
		st.dsn,
		DefaultReconnectionTimeout,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				fmt.Println(err.Error())
			}
		},
	)

	// TODO: proper error handling!
	defer listener.Close()

	if err := listener.Listen(name); err != nil {
		return fmt.Errorf("postgres.EventStore: failed to listen on stream: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres.EventStore: listener closed: %w", ctx.Err())

		case <-time.After(DefaultNotifyChannelTimeout):
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
					Payload:  vp.Elem().Interface().(eventually.Payload),
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
			WHERE stream_type = $1 AND global_sequence_number >= $2
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
			WHERE stream_type = $1 AND stream_id = $2 AND "version" >= $3
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

		event.Payload = vp.Elem().Interface().(eventually.Payload)
		event.Metadata = metadata
		event.Event = event.WithGlobalSequenceNumber(globalSequenceNumber)

		es <- event
	}

	return nil
}

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

// ErrEmptyEventsMap occurs during a call to Register where a nil or empty Events map
// is provided, which would mean no events would be registered for the desired type.
var ErrEmptyEventsMap = fmt.Errorf("postgres.EventStore: empty events map provided for type")

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

// Register registers Domain Events used by the application in order to decode events
// stored in the database by their name returned by the eventually.Payload trait.
func (st *EventStore) Register(ctx context.Context, events ...eventually.Payload) error {
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
		if event == nil {
			return fmt.Errorf("postgres.EventStore: expected event type, nil was provided instead")
		}

		eventName := event.Name()
		eventType := reflect.TypeOf(event)

		if registeredType, ok := st.eventNameToType[eventName]; ok {
			// TODO(ar3s3ru): this is a clear code smell for the current Event Store API.
			// We can find a different way of registering events.
			if registeredType == eventType {
				// Type is already registered and the new one is the same as the
				// one already registered, so we can continue with the other event types.
				continue
			}

			return fmt.Errorf(
				"postgres.EventStore: event '%s' has been already registered with a different type",
				eventName,
			)
		}

		st.eventNameToType[eventName] = eventType
		st.eventTypeToName[eventType] = eventName
	}

	return nil
}

// StreamAll opens an Event Stream and sinks all the events in the Event Store in the provided
// channel, skipping those events with a sequence number lower than the provided bound.
func (st *EventStore) StreamAll(ctx context.Context, es eventstore.EventStream, selectt eventstore.Select) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE global_sequence_number >= $1
			ORDER BY global_sequence_number ASC`,
		selectt.From,
	)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	return st.rowsToStream(rows, es)
}

func (st *EventStore) StreamByType(
	ctx context.Context,
	es eventstore.EventStream,
	typ string,
	selectt eventstore.Select,
) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE stream_type = $1 AND global_sequence_number >= $2
			ORDER BY global_sequence_number ASC`,
		typ,
		selectt.From,
	)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	return st.rowsToStream(rows, es)
}

func (st *EventStore) Stream(
	ctx context.Context,
	es eventstore.EventStream,
	id eventstore.StreamID,
	selectt eventstore.Select,
) error {
	defer close(es)

	rows, err := st.db.QueryContext(
		ctx,
		`SELECT * FROM events
			WHERE stream_type = $1 AND stream_id = $2 AND "version" >= $3
			ORDER BY "version" ASC`,
		id.Type,
		id.Name,
		selectt.From,
	)
	if err != nil {
		return fmt.Errorf("postgres.EventStore: failed to get events from store: %w", err)
	}

	return st.rowsToStream(rows, es)
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
			&event.StreamID.Type,
			&event.StreamID.Name,
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

type rawNotificationEvent struct {
	StreamID   string              `json:"stream_id"`
	StreamType string              `json:"stream_type"`
	EventType  string              `json:"event_type"`
	Version    int64               `json:"version"`
	Event      json.RawMessage     `json:"event"`
	Metadata   eventually.Metadata `json:"metadata"`
}

// SubscribeToAll subscribes to all the new Events committed to the Event Store
// and sinks them in the provided channel.
func (st *EventStore) SubscribeToAll(ctx context.Context, es eventstore.EventStream) error {
	return st.subscribe(ctx, streamAllName, es)
}

func (st *EventStore) SubscribeToType(ctx context.Context, es eventstore.EventStream, typ string) error {
	return st.subscribe(ctx, typ, es)
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

			event, err := st.processNotification(notification)
			if err != nil {
				return err
			}

			es <- event
		}
	}
}

func (st *EventStore) processNotification(notification *pq.Notification) (eventstore.Event, error) {
	buffer := bytes.NewBufferString(notification.Extra)

	var rawEvent rawNotificationEvent
	if err := json.NewDecoder(buffer).Decode(&rawEvent); err != nil {
		return eventstore.Event{}, fmt.Errorf(
			"postgres.EventStore: failed to unmarshal notification payload into event: %w",
			err,
		)
	}

	payload, ok := st.eventNameToType[rawEvent.EventType]
	if !ok {
		return eventstore.Event{}, fmt.Errorf(
			"postgres.EventStore: received unregistered event '%s'",
			rawEvent.EventType,
		)
	}

	vp := reflect.New(payload)
	if err := json.Unmarshal(rawEvent.Event, vp.Interface()); err != nil {
		return eventstore.Event{}, fmt.Errorf(
			"postgres.EventStore: failed to unmarshal event payload from json: %w",
			err,
		)
	}

	return eventstore.Event{
		StreamID: eventstore.StreamID{
			Type: rawEvent.StreamType,
			Name: rawEvent.StreamID,
		},
		Version: rawEvent.Version,
		Event: eventually.Event{
			Payload:  vp.Elem().Interface().(eventually.Payload),
			Metadata: rawEvent.Metadata,
		},
	}, nil
}

func (st *EventStore) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (v int64, err error) {
	tx, err := st.db.BeginTx(ctx, nil)
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
		if v, err = st.appendEvent(ctx, tx, id, expected, event); err != nil {
			return 0, err
		}

		// Update the expected version for the next event with the new version.
		expected = eventstore.VersionCheck(v)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to commit append transaction: %w", err)
	}

	return v, nil
}

func (st *EventStore) appendEvent(
	ctx context.Context,
	tx *sql.Tx,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	event eventually.Event,
) (int64, error) {
	eventType := reflect.TypeOf(event.Payload)
	eventName, ok := st.eventTypeToName[eventType]

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

	var newVersion int64

	err = tx.QueryRowContext(
		ctx,
		"SELECT append_to_store($1, $2, $3, $4, $5, $6)",
		id.Type,
		id.Name,
		int64(expected),
		eventName,
		eventPayload,
		metadata,
	).Scan(&newVersion)

	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to append event: %w", err)
	}

	return newVersion, nil
}

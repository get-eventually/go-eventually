package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"

	_ "github.com/golang-migrate/migrate/database/postgres" // postgres driver for migrate
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

var _ eventstore.Store = &EventStore{}

// EventStore is an eventstore.Store implementation which uses
// PostgreSQL as backend datastore.
type EventStore struct {
	dsn      string
	db       *sql.DB
	registry eventstore.Registry
}

// OpenEventStore opens a connection with the PostgreSQL identified by the provided DSN.
// Make sure to perform migrations first by running postgres.RunMigrations() function.
func OpenEventStore(dsn string) (*EventStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.EventStore: failed to open connection with the db: %w", err)
	}

	registry, err := eventstore.NewRegistry(json.Unmarshal)
	if err != nil {
		return nil, fmt.Errorf("postgres.EventStore: failed to initialize event registry: %w", err)
	}

	return &EventStore{
		db:       db,
		dsn:      dsn,
		registry: registry,
	}, nil
}

// Close closes the Event Store database connection.
func (st *EventStore) Close() error {
	return st.db.Close()
}

// Register registers Domain Events used by the application in order to decode events
// stored in the database by their name returned by the eventually.Payload trait.
func (st *EventStore) Register(events ...eventually.Payload) error {
	// TODO(ar3s3ru): get the registry injected instead of creating it in OpenEventStore().
	return st.registry.Register(events...)
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

	// FIXME(ar3s3ru): add logger support in the event store
	return rowsToStream(rows, es, st.registry, nil)
}

// StreamByType opens a stream of all Event Streams grouped by the same Type,
// as specified in input.
//
// The stream will be ordered based on their Global Sequence Number.
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

	// FIXME(ar3s3ru): add logger support in the event store
	return rowsToStream(rows, es, st.registry, nil)
}

// Stream opens the specific Event Stream identified by the provided id.
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

	// FIXME(ar3s3ru): add logger support in the event store
	return rowsToStream(rows, es, st.registry, nil)
}

type rawNotificationEvent struct {
	StreamID       string              `json:"stream_id"`
	StreamType     string              `json:"stream_type"`
	EventType      string              `json:"event_type"`
	Version        int64               `json:"version"`
	SequenceNumber int64               `json:"sequence_number"`
	Event          json.RawMessage     `json:"event"`
	Metadata       eventually.Metadata `json:"metadata"`
}

// SubscribeToAll subscribes to all the new Events committed to the Event Store
// and sinks them in the provided channel.
func (st *EventStore) SubscribeToAll(ctx context.Context, es eventstore.EventStream) error {
	return st.subscribe(ctx, streamAllName, es)
}

// SubscribeToType subscribes to all the new Events of the specified Stream Type
// committed to the Event Store and sinks them in the provided channel.
func (st *EventStore) SubscribeToType(ctx context.Context, es eventstore.EventStream, typ string) error {
	return st.subscribe(ctx, typ, es)
}

func (st *EventStore) subscribe(ctx context.Context, name string, es eventstore.EventStream) (err error) {
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

	if err = listener.Listen(name); err != nil {
		return fmt.Errorf("postgres.EventStore: failed to listen on stream: %w", err)
	}

	// TODO: proper error handling!
	defer func() {
		//nolint:errcheck,gosec // Skipping proper error handling for now, as it would require
		//                         logging and we haven't set that up yet in here.
		listener.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres.EventStore: listener closed: %w", ctx.Err())

		case <-time.After(DefaultNotifyChannelTimeout):
			if err = listener.Ping(); err != nil {
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

	payload, err := st.registry.Deserialize(rawEvent.EventType, rawEvent.Event)
	if err != nil {
		return eventstore.Event{}, fmt.Errorf(
			"postgres.EventStore: failed to unmarshal event payload from json: %w",
			err,
		)
	}

	return eventstore.Event{
		Stream: eventstore.StreamID{
			Type: rawEvent.StreamType,
			Name: rawEvent.StreamID,
		},
		Version:        rawEvent.Version,
		SequenceNumber: rawEvent.SequenceNumber,
		Event: eventually.Event{
			Payload:  payload,
			Metadata: rawEvent.Metadata,
		},
	}, nil
}

// Append inserts the specified Domain Events into the Event Stream specified
// by the current instance, returning the new version of the Event Stream.
//
// A version can be specified to enable an Optimistic Concurrency check
// on append, by using the expected version of the Event Stream prior
// to appending the new Events.
//
// Alternatively, VersionCheckAny can be used if no Optimistic Concurrency check
// should be carried out.
//
// NOTE: this implementation is not returning yet eventstore.ErrConflict in case
// of conflicting expectations with the provided VersionCheck value.
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

// TODO(ar3s3ru): add the ErrConflict error in case of optimistic concurrency issues.
func (st *EventStore) appendEvent(
	ctx context.Context,
	tx *sql.Tx,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	event eventually.Event,
) (int64, error) {
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
		event.Payload.Name(),
		eventPayload,
		metadata,
	).Scan(&newVersion)

	if err != nil {
		return 0, fmt.Errorf("postgres.EventStore: failed to append event: %w", err)
	}

	return newVersion, nil
}

package messagedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/get-eventually/go-eventually/extension/correlation"
	"reflect"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
)

// ErrEmptyEventsMap occurs during a call to Register where a nil or empty Events map
// is provided, which would mean no events would be registered for the desired type.
var ErrEmptyEventsMap = fmt.Errorf("messagedb.EventStore: empty events map provided for type")

var (
	_ eventstore.Appender = &EventStore{}
	_ eventstore.Streamer = &EventStore{}
)

type EventStore struct {
	DB *sql.DB

	eventNameToType map[string]reflect.Type
	eventTypeToName map[reflect.Type]string
}

// Register registers Domain Events used by the application in order to decode events
// stored in the database by their name returned by the eventually.Payload trait.
func (s *EventStore) Register(ctx context.Context, events ...eventually.Payload) error {
	if len(events) == 0 {
		return ErrEmptyEventsMap
	}

	if err := s.registerEventsToType(events...); err != nil {
		return fmt.Errorf("messagedb.EventStore: failed to register types: %w", err)
	}

	return nil
}

func (s *EventStore) registerEventsToType(events ...eventually.Payload) error {
	for _, event := range events {
		if event == nil {
			return fmt.Errorf("messagedb.EventStore: expected event type, nil was provided instead")
		}

		eventName := event.Name()
		eventType := reflect.TypeOf(event)

		if registeredType, ok := s.eventNameToType[eventName]; ok {
			// TODO(ar3s3ru): this is a clear code smell for the current Event Store API.
			// We can find a different way of registering events.
			if registeredType == eventType {
				// Type is already registered and the new one is the same as the
				// one already registered, so we can continue with the other event types.
				continue
			}

			return fmt.Errorf(
				"messagedb.EventStore: event '%s' has been already registered with a different type",
				eventName,
			)
		}

		s.eventNameToType[eventName] = eventType
		s.eventTypeToName[eventType] = eventName
	}

	return nil
}

func (s *EventStore) Append(
	ctx context.Context,
	id eventstore.StreamID,
	check eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	/*
	write_message(
	  id varchar,
	  stream_name varchar,
	  type varchar,
	  data jsonb,
	  metadata jsonb DEFAULT NULL,
	  expected_version bigint DEFAULT NULL
	)
	 */

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("messagedb.EventStore: failed to start a transaction: %w", err)
	}

	streamName := fmt.Sprintf("%s-%s", id.Type, id.Name)

	for _, e := range events {
		eventId, ok := correlation.Message(e).EventID()
		if !ok {
			eventId = "todo"
		}

		payload, err := json.Marshal(e.Payload)
		if err != nil {
			return 0, fmt.Errorf("messagedb.EventStore: failed to serialize event payload: %w", err)
		}

		metadata, err := json.Marshal(e.Metadata)
		if err != nil {
			return 0, fmt.Errorf("messagedb.EventStore: failed to serialize event metadata: %w", err)
		}

		var expectedVersion interface{}

		if check != eventstore.VersionCheckAny {
			expectedVersion = int64(check)
		}

		rows, err := tx.QueryContext(ctx, "select write_message($1, $2, $3, $4, $5, $6)",
			eventId,
			streamName,
			e.Payload.Name(),
			payload,
			metadata,
			expectedVersion,
		)
		if err != nil {
			return 0, fmt.Errorf("messagedb.EventStore: failed to append event: %w", err)
		}

		var position int64
		err = rows.Scan(&position)

		_ = rows.Close()
		//todo: log error

		if err != nil {
			return 0, fmt.Errorf("messagedb.EventStore: failed to read result of appending an event: %w", err)
		}
	}
	
	//todo: continue
}

func (s *EventStore) Stream(
	ctx context.Context,
	es eventstore.EventStream,
	id eventstore.StreamID,
	selectt eventstore.Select,
) error {
	panic("implement me")
}

func (s *EventStore) StreamByType(
	ctx context.Context,
	es eventstore.EventStream,
	streamType string,
	selectt eventstore.Select,
) error {
	panic("implement me")
}

func (s *EventStore) StreamAll(
	ctx context.Context,
	es eventstore.EventStream,
	selectt eventstore.Select,
) error {
	panic("implement me")
}

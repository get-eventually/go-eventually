package messagedb

import (
	"context"
	"database/sql"
	"fmt"
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
func (st *EventStore) Register(ctx context.Context, events ...eventually.Payload) error {
	if len(events) == 0 {
		return ErrEmptyEventsMap
	}

	if err := st.registerEventsToType(events...); err != nil {
		return fmt.Errorf("messagedb.EventStore: failed to register types: %w", err)
	}

	return nil
}

func (st *EventStore) registerEventsToType(events ...eventually.Payload) error {
	for _, event := range events {
		if event == nil {
			return fmt.Errorf("messagedb.EventStore: expected event type, nil was provided instead")
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
				"messagedb.EventStore: event '%s' has been already registered with a different type",
				eventName,
			)
		}

		st.eventNameToType[eventName] = eventType
		st.eventTypeToName[eventType] = eventName
	}

	return nil
}

func (s *EventStore) Append(
	ctx context.Context,
	id eventstore.StreamID,
	check eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	panic("implement me")
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

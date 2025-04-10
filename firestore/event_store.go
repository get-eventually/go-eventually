package firestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

//nolint:exhaustruct // Only used for interface assertion.
var _ event.Store = EventStore{}

// EventStore is an event.Store implementation using Google Cloud Firestore as backend.
type EventStore struct {
	Client *firestore.Client
	Serde  serde.Bytes[message.Message]
}

// NewEventStore creates a new EventStore instance.
func NewEventStore(client *firestore.Client, msgSerde serde.Bytes[message.Message]) EventStore {
	return EventStore{Client: client, Serde: msgSerde}
}

func (es EventStore) eventsCollection() *firestore.CollectionRef {
	return es.Client.Collection("Events")
}

func (es EventStore) streamsCollection() *firestore.CollectionRef {
	return es.Client.Collection("EventStreams")
}

// Stream implements the event.Streamer interface.
func (es EventStore) Stream(
	ctx context.Context,
	stream event.StreamWrite,
	id event.StreamID,
	selector version.Selector,
) error {
	defer close(stream)

	iter := es.eventsCollection().
		Where("event_stream_id", "==", string(id)).
		Where("version", ">=", selector.From).
		OrderBy("version", firestore.Asc).
		Documents(ctx)

	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return fmt.Errorf("firestore.EventStore.Stream: failed while reading iterator, %w", err)
		}

		payload, ok := doc.Data()["payload"].([]byte)
		if !ok {
			return fmt.Errorf("firestore.EventStore.Stream: invalid payload type, expected: []byte, got: %T", doc.Data()["payload"])
		}

		msg, err := es.Serde.Deserialize(payload)
		if err != nil {
			return fmt.Errorf("firestore.EventStore.Stream: failed to deserialize message payload, %w", err)
		}

		var metadata message.Metadata
		if v, ok := doc.Data()["metadata"].(message.Metadata); ok && v != nil {
			metadata = v
		}

		v, ok := doc.Data()["version"].(int64)
		if !ok {
			return fmt.Errorf("firestore.EventStore.Stream: invalid version type, expected: int64, got: %T", doc.Data()["version"])
		}

		stream <- event.Persisted{
			StreamID: id,
			Version:  version.Version(v), //nolint:gosec // This should not overflow.
			Envelope: event.Envelope{
				Message:  msg,
				Metadata: metadata,
			},
		}
	}

	return nil
}

func (es EventStore) checkAndUpsertEventStream(
	tx *firestore.Transaction,
	id event.StreamID,
	expected version.Check,
	newEventsLength int,
) (version.Version, error) {
	docRef := es.streamsCollection().Doc(string(id))

	doc, err := tx.Get(docRef)
	if err != nil && status.Code(err) != codes.NotFound {
		return 0, fmt.Errorf("firestore.EventStore.Append: failed to get stream, %w", err)
	}

	var currentVersion version.Version

	if err == nil {
		lastVersion, ok := doc.Data()["last_version"].(int64)
		if !ok {
			return 0, fmt.Errorf("firestore.EventStore.Append: invalid last_version type, expected: int64, got: %T", doc.Data()["last_version"])
		}

		currentVersion = version.Version(lastVersion) //nolint:gosec // This should not overflow.
	}

	if v, ok := expected.(version.CheckExact); ok && version.Version(v) != currentVersion {
		return 0, fmt.Errorf("firestore.EventStore.Append: version check failed, %w", version.ConflictError{
			Expected: version.Version(v),
			Actual:   currentVersion,
		})
	}

	newVersion := currentVersion + version.Version(newEventsLength) //nolint:gosec // This should not overflow.

	if err := tx.Set(docRef, map[string]interface{}{
		"last_version": newVersion,
	}); err != nil {
		return 0, fmt.Errorf("firestore.EventStore.Append: failed to update event stream, %w", err)
	}

	return currentVersion, nil
}

func (es EventStore) appendEvent(tx *firestore.Transaction, evt event.Persisted) error {
	id := fmt.Sprintf("%s@{%d}", evt.StreamID, evt.Version)
	docRef := es.eventsCollection().Doc(id)

	payload, err := es.Serde.Serialize(evt.Message)
	if err != nil {
		return fmt.Errorf("firestore.EventStore.appendEvent: failed to serialize message, %w", err)
	}

	if err := tx.Create(docRef, map[string]interface{}{
		"event_stream_id": string(evt.StreamID),
		"version":         evt.Version,
		"type":            evt.Message.Name(),
		"metadata":        evt.Metadata,
		"payload":         payload,
	}); err != nil {
		return fmt.Errorf("firestore.EventStore.appendEvent: failed to append event, %w", err)
	}

	return nil
}

// Append implements the event.Appender interface.
func (es EventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	var currentVersion version.Version

	if err := es.Client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		var err error

		if currentVersion, err = es.checkAndUpsertEventStream(tx, id, expected, len(events)); err != nil {
			return err
		}

		for i, evt := range events {
			if err := es.appendEvent(tx, event.Persisted{
				StreamID: id,
				Version:  currentVersion + version.Version(i) + 1, //nolint:gosec // This should not overflow.
				Envelope: evt,
			}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return 0, fmt.Errorf("firestore.EventStore.Append: failed to commit transaction, %w", err)
	}

	return currentVersion + version.Version(len(events)), nil //nolint:gosec // This should not overflow.
}

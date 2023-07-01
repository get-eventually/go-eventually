package eventuallyfirestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

//nolint:exhaustruct // Only used for interface assertion.
var _ event.Store = EventStore{}

type EventStore struct {
	Client *firestore.Client
	Serde  serde.Bytes[message.Message]
}

func (es EventStore) eventsCollection() *firestore.CollectionRef {
	return es.Client.Collection("Events")
}

func (es EventStore) streamsCollection() *firestore.CollectionRef {
	return es.Client.Collection("EventStreams")
}

func printDocs(prefix string, documents []*firestore.DocumentSnapshot) {
	printable := make([]map[string]interface{}, 0, len(documents))
	for _, v := range documents {
		printable = append(printable, v.Data())
	}

	fmt.Printf("PRINT %s: %#v\n\n", prefix, printable)
}

// Stream implements the event.Streamer interface.
func (es EventStore) Stream(
	ctx context.Context,
	stream event.StreamWrite,
	id event.StreamID,
	selector version.Selector,
) error {
	defer close(stream)

	docs, _ := es.eventsCollection().Documents(ctx).GetAll()
	printDocs("EVENTS", docs)

	docs, _ = es.streamsCollection().Documents(ctx).GetAll()
	printDocs("STREAMS", docs)

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
			return fmt.Errorf("eventuallyfirestore.EventStore.Stream: failed while reading iterator, %w", err)
		}

		msg, err := es.Serde.Deserialize(doc.Data()["payload"].([]byte))
		if err != nil {
			return fmt.Errorf("eventuallyfirestore.EventStore.Stream: failed to deserialize message payload, %w", err)
		}

		var metadata message.Metadata
		if v, ok := doc.Data()["metadata"]; ok && v != nil {
			metadata = v.(message.Metadata)
		}

		stream <- event.Persisted{
			StreamID: id,
			Version:  version.Version(doc.Data()["version"].(int64)),
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
		return 0, fmt.Errorf("eventuallyfirestore.EventStore.Append: failed to get stream, %w", err)
	}

	var currentVersion version.Version
	if err == nil {
		currentVersion = version.Version(doc.Data()["last_version"].(int64))
	}

	if v, ok := expected.(version.CheckExact); ok && version.Version(v) != currentVersion {
		return 0, fmt.Errorf("eventuallyfirestore.EventStore.Append: version check failed, %w", version.ConflictError{
			Expected: version.Version(v),
			Actual:   currentVersion,
		})
	}

	newVersion := currentVersion + version.Version(newEventsLength)

	if err := tx.Set(docRef, map[string]interface{}{
		"last_version": newVersion,
	}); err != nil {
		return 0, fmt.Errorf("eventuallyfirestore.EventStore.Append: failed to update event stream, %w", err)
	}

	return currentVersion, nil
}

func (es EventStore) appendEvent(tx *firestore.Transaction, evt event.Persisted) error {
	id := fmt.Sprintf("%s@{%d}", evt.StreamID, evt.Version)
	docRef := es.eventsCollection().Doc(id)

	payload, err := es.Serde.Serialize(evt.Message)
	if err != nil {
		return fmt.Errorf("eventuallyfirestore.EventStore.appendEvent: failed to serialize message, %w", err)
	}

	if err := tx.Create(docRef, map[string]interface{}{
		"event_stream_id": string(evt.StreamID),
		"version":         evt.Version,
		"type":            evt.Message.Name(),
		"metadata":        evt.Metadata,
		"payload":         payload,
	}); err != nil {
		return fmt.Errorf("eventuallyfirestore.EventStore.appendEvent: failed to append event, %w", err)
	}

	return nil
}

func (es EventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	var currentVersion version.Version

	err := es.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var err error

		currentVersion, err = es.checkAndUpsertEventStream(tx, id, expected, len(events))
		if err != nil {
			return err
		}

		for i, evt := range events {
			if err := es.appendEvent(tx, event.Persisted{
				StreamID: id,
				Version:  currentVersion + version.Version(i) + 1,
				Envelope: evt,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("eventuallyfirestore.EventStore.Append: failed to commit transaction, %w", err)
	}

	return currentVersion + version.Version(len(events)), nil
}

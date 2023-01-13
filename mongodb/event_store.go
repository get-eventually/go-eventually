package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/core/version"
)

var _ event.Store = EventStore{}

type EventStore struct {
	Client       *mongo.Client
	DatabaseName string
	Serde        serde.Bytes[message.Message]
}

func (es EventStore) openSession() (mongo.Session, error) {
	return es.Client.StartSession(&options.SessionOptions{
		DefaultReadConcern:    readconcern.Majority(),
		DefaultReadPreference: readpref.Primary(),
		// Prefer that the write operations, being transactions with strong validations,
		// are replicated to the majority of the replicas in the replica set.
		DefaultWriteConcern: writeconcern.New(
			writeconcern.WMajority(),
		),
	})
}

func (es EventStore) database() *mongo.Database {
	return es.Client.Database(es.DatabaseName, &options.DatabaseOptions{
		// If we're using event.Store.Stream, is to re-hydrate the state of
		// an Aggregate to perform a write operation through a Command.
		ReadConcern:    readconcern.Majority(),
		ReadPreference: readpref.Primary(),
	})
}

func (es EventStore) eventsCollection() *mongo.Collection {
	return es.database().Collection("events")
}

func (es EventStore) eventStreamsCollection() *mongo.Collection {
	return es.database().Collection("event_streams")
}

// updateEventStream updates the Event Stream entry in the `event_streams`
// collection and performs optimistic locking checks.
//
// Returns the old version of the Event Stream, before the update.
func (es EventStore) updateEventStream(
	ctx mongo.SessionContext,
	id event.StreamID,
	expected version.Check,
	newVersionOffset int,
) (version.Version, error) {
	eventStreamsCollection := es.eventStreamsCollection()

	var eventStream bson.M

	err := eventStreamsCollection.
		FindOne(ctx, bson.D{{Key: "_id", Value: string(id)}}).
		Decode(&eventStream)

	if errors.Is(err, mongo.ErrNoDocuments) {
		eventStream = bson.M{"_id": string(id), "version": int64(0)}
	} else if err != nil {
		return 0, fmt.Errorf("mongodb.EventStore: failed to find event stream, %w", err)
	}

	currentVersion := version.Version(eventStream["version"].(int64))
	if v, ok := expected.(version.CheckExact); ok && currentVersion != version.Version(v) {
		return 0, version.ConflictError{
			Expected: version.Version(v),
			Actual:   currentVersion,
		}
	}

	newVersion := currentVersion + version.Version(newVersionOffset)
	eventStream["version"] = newVersion

	panic("implement me!")
}

func (es EventStore) append(
	ctx mongo.SessionContext,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	oldEventStreamVersion, err := es.updateEventStream(ctx, id, expected, len(events))
	if err != nil {
		return 0, fmt.Errorf("mongodb.EventStore: failed to update event stream version, %w", err)
	}

	eventsCollection := es.eventsCollection()

	var documents bson.A
	for i, evt := range events {
		msg, err := es.Serde.Serialize(evt.Message)
		if err != nil {
			return 0, fmt.Errorf("mongodb.EventStore: failed to serialize event, %w", err)
		}

		documents = append(documents, bson.M{
			"event_stream_id": string(id),
			"version":         uint64(oldEventStreamVersion) + uint64(i) + 1,
			"message":         msg,
			"metadata":        evt.Metadata,
		})
	}

	if _, err := eventsCollection.InsertMany(ctx, documents); err != nil {
		return 0, fmt.Errorf("mongodb.EventStore: failed to insert new domain events, %w", err)
	}

	panic("implement me!")
}

// Append implements event.Store
func (es EventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	sess, err := es.openSession()
	if err != nil {
		return 0, fmt.Errorf("mongodb.EventStore: failed to open a new session, %w", err)
	}

	result, err := sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return es.append(sessCtx, id, expected, events...)
	})

	return result.(version.Version), err
}

// Stream implements event.Store
func (es EventStore) Stream(
	ctx context.Context,
	stream event.StreamWrite,
	id event.StreamID,
	selector version.Selector,
) error {
	defer close(stream)

	eventsCollection := es.eventsCollection()
	cursor, err := eventsCollection.Find(ctx, bson.D{
		{
			Key:   "event_stream_id",
			Value: string(id),
		},
		{
			Key: "version",
			Value: bson.D{
				{
					Key:   "$gte",
					Value: selector.From,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("mongodb.EventStore: failed to open event stream cursor, %w", err)
	}

	for cursor.Next(ctx) {
		// TODO: write the conversion logic from document to event message.
		// We can register a common type and use a bsoncodec.Registry for passing struct values directly.
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("mongodb.EventStore: failed while iterating the event stream query cursor, %w", err)
	}

	return nil
}

package inmemory

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

// TrackingEventStore is an Event Store wrapper to track the Events
// committed to the inner Event Store.
//
// Useful for tests assertion.
type TrackingEventStore struct {
	eventstore.Store
	recorded []eventstore.Event
}

// Recorded returns the list of recorded Events.
func (es *TrackingEventStore) Recorded() []eventstore.Event { return es.recorded }

// Type returns an eventstore.Typed instance for the specified type identifier.
func (es *TrackingEventStore) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	typed, err := es.Store.Type(ctx, typ)
	if err != nil {
		return typed, err
	}

	return &typedTrackingEventStore{
		parent:     es,
		streamType: typ,
		Typed:      typed,
	}, nil
}

type typedTrackingEventStore struct {
	eventstore.Typed
	streamType string
	parent     *TrackingEventStore
}

func (es *typedTrackingEventStore) Instance(id string) eventstore.Instanced {
	return &instancedTrackingEventStore{
		parent:     es.parent,
		streamName: id,
		streamType: es.streamType,
		Instanced:  es.Typed.Instance(id),
	}
}

type instancedTrackingEventStore struct {
	eventstore.Instanced
	streamType string
	streamName string
	parent     *TrackingEventStore
}

func (es *instancedTrackingEventStore) Append(
	ctx context.Context,
	version int64,
	events ...eventually.Event,
) (int64, error) {
	v, err := es.Instanced.Append(ctx, version, events...)
	if err != nil {
		return v, err
	}

	for i, event := range events {
		es.parent.recorded = append(es.parent.recorded, eventstore.Event{
			StreamType: es.streamType,
			StreamName: es.streamName,
			Version:    version + int64(i) + 1,
			Event:      event,
		})
	}

	return v, err
}

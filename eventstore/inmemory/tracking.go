package inmemory

import (
	"context"
	"sync"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

// TrackingEventStore is an Event Store wrapper to track the Events
// committed to the inner Event Store.
//
// Useful for tests assertion.
type TrackingEventStore struct {
	sync.RWMutex

	eventStore eventstore.Store
	recorded   []eventstore.Event
}

func NewTrackingEventStore(store eventstore.Store) *TrackingEventStore {
	return &TrackingEventStore{eventStore: store}
}

func (es *TrackingEventStore) Recorded() []eventstore.Event {
	es.RLock()
	defer es.RUnlock()

	return es.recorded[:]
}

func (es *TrackingEventStore) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	es.Lock()
	defer es.Unlock()

	v, err := es.eventStore.Append(ctx, id, expected, events...)
	if err != nil {
		return v, err
	}

	for i, event := range events {
		es.recorded = append(es.recorded, eventstore.Event{
			StreamID: id,
			Version:  int64(expected) + int64(i) + 1,
			Event:    event,
		})
	}

	return v, err
}

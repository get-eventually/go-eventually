package inmemory

import (
	"context"
	"sync"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
)

// TrackingEventStore is an Event Store wrapper to track the Events
// committed to the inner Event Store.
//
// Useful for tests assertion.
type TrackingEventStore struct {
	sync.RWMutex

	// TODO(ar3s3ru): we can use eventstore.Appender here instead
	eventstore.Store
	recorded []eventstore.Event
}

// NewTrackingEventStore wraps an Event Store to capture events that get
// appended to it.
func NewTrackingEventStore(store eventstore.Store) *TrackingEventStore {
	return &TrackingEventStore{Store: store}
}

// Recorded returns the list of Events that have been appended
// to the Event Store.
//
// Please note: these events do not record the Sequence Number assigned by
// the Event Store. Usually you should not need it in test assertions, since
// the order of Events in the returned slice always follows the global order
// of the Event Stream (or the Event Store).
func (es *TrackingEventStore) Recorded() []eventstore.Event {
	es.RLock()
	defer es.RUnlock()

	return es.recorded
}

// Append forwards the call to the wrapped Event Store instance and,
// if the operation concludes successfully, records these events internally.
//
// The recorded events can be accessed by calling Recorded().
func (es *TrackingEventStore) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	es.Lock()
	defer es.Unlock()

	v, err := es.Store.Append(ctx, id, expected, events...)
	if err != nil {
		return v, err
	}

	for i, event := range events {
		es.recorded = append(es.recorded, eventstore.Event{
			Stream:  id,
			Version: int64(expected) + int64(i) + 1,
			Event:   event,
		})
	}

	return v, err
}

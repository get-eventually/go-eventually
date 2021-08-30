package inmemory

import (
	"context"
	"sync"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

// TrackingEventStore is an Event Store wrapper to track the Events
// committed to the inner Event Store.
//
// Useful for tests assertion.
type TrackingEventStore struct {
	eventstore.Appender

	mx       sync.RWMutex
	recorded []eventstore.Event
}

// NewTrackingEventStore wraps an Event Store to capture events that get
// appended to it.
func NewTrackingEventStore(appender eventstore.Appender) *TrackingEventStore {
	return &TrackingEventStore{Appender: appender}
}

// Recorded returns the list of Events that have been appended
// to the Event Store.
//
// Please note: these events do not record the Sequence Number assigned by
// the Event Store. Usually you should not need it in test assertions, since
// the order of Events in the returned slice always follows the global order
// of the Event Stream (or the Event Store).
func (es *TrackingEventStore) Recorded() []eventstore.Event {
	es.mx.RLock()
	defer es.mx.RUnlock()

	return es.recorded
}

// Append forwards the call to the wrapped Event Store instance and,
// if the operation concludes successfully, records these events internally.
//
// The recorded events can be accessed by calling Recorded().
func (es *TrackingEventStore) Append(
	ctx context.Context,
	id stream.ID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	es.mx.Lock()
	defer es.mx.Unlock()

	v, err := es.Appender.Append(ctx, id, expected, events...)
	if err != nil {
		return v, err
	}

	previousVersion := v - int64(len(events))

	for i, event := range events {
		es.recorded = append(es.recorded, eventstore.Event{
			Stream:  id,
			Version: previousVersion + int64(i) + 1,
			Event:   event,
		})
	}

	return v, err
}

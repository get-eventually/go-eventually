package inmemory

import (
	"context"
	"sync"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/version"
)

// TrackingEventStore is an Event Store wrapper to track the Events
// committed to the inner Event Store.
//
// Useful for tests assertion.
type TrackingEventStore struct {
	event.Appender

	mx       sync.RWMutex
	recorded []event.Persisted
}

// NewTrackingEventStore wraps an Event Store to capture events that get
// appended to it.
func NewTrackingEventStore(appender event.Appender) *TrackingEventStore {
	return &TrackingEventStore{Appender: appender}
}

// Recorded returns the list of Events that have been appended
// to the Event Store.
//
// Please note: these events do not record the Sequence Number assigned by
// the Event Store. Usually you should not need it in test assertions, since
// the order of Events in the returned slice always follows the global order
// of the Event Stream (or the Event Store).
func (es *TrackingEventStore) Recorded() []event.Persisted {
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
	id event.StreamID,
	expected version.Check,
	events ...event.Event,
) (version.Version, error) {
	es.mx.Lock()
	defer es.mx.Unlock()

	v, err := es.Appender.Append(ctx, id, expected, events...)
	if err != nil {
		return v, err
	}

	previousVersion := v - version.Version(len(events))

	for i, evt := range events {
		es.recorded = append(es.recorded, event.Persisted{
			Stream:  id,
			Version: previousVersion + version.Version(i) + 1,
			Event:   evt,
		})
	}

	return v, err
}

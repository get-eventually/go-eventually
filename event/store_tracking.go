package event

import (
	"context"
	"sync"

	"github.com/get-eventually/go-eventually/version"
)

// TrackingStore is an Event Store wrapper to track the Events
// committed to the inner Event Store.
//
// Useful for tests assertion.
type TrackingStore struct {
	Appender

	mx       sync.RWMutex
	recorded []Persisted
}

// NewTrackingStore wraps an Event Store to capture events that get
// appended to it.
func NewTrackingStore(appender Appender) *TrackingStore {
	return &TrackingStore{
		Appender: appender,
		mx:       sync.RWMutex{},
		recorded: nil,
	}
}

// Recorded returns the list of Events that have been appended
// to the Event Store.
//
// Please note: these events do not record the Sequence Number assigned by
// the Event Store. Usually you should not need it in test assertions, since
// the order of Events in the returned slice always follows the global order
// of the Event Stream (or the Event Store).
func (es *TrackingStore) Recorded() []Persisted {
	es.mx.RLock()
	defer es.mx.RUnlock()

	return es.recorded
}

// Append forwards the call to the wrapped Event Store instance and,
// if the operation concludes successfully, records these events internally.
//
// The recorded events can be accessed by calling Recorded().
func (es *TrackingStore) Append(
	ctx context.Context,
	id StreamID,
	expected version.Check,
	events ...Envelope,
) (version.Version, error) {
	es.mx.Lock()
	defer es.mx.Unlock()

	v, err := es.Appender.Append(ctx, id, expected, events...)
	if err != nil {
		return v, err
	}

	previousVersion := v - version.Version(len(events))

	for i, evt := range events {
		es.recorded = append(es.recorded, Persisted{
			StreamID: id,
			Version:  previousVersion + version.Version(i) + 1,
			Envelope: evt,
		})
	}

	return v, err
}

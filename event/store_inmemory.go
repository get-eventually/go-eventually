package event

import (
	"context"
	"fmt"
	"sync"

	"github.com/get-eventually/go-eventually/version"
)

// Interface implementation assertion.
var _ Store = new(InMemoryStore)

// InMemoryStore is a thread-safe, in-memory event.Store implementation.
type InMemoryStore struct {
	mx     sync.RWMutex
	events map[StreamID][]Envelope
}

// NewInMemoryStore creates a new event.InMemoryStore instance.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		mx:     sync.RWMutex{},
		events: make(map[StreamID][]Envelope),
	}
}

// Stream returns a Stream over the committed events for the given Event Stream,
// filtered by the provided version.Selector.
//
// The returned Stream holds a read-lock on the underlying store for the
// duration of iteration; long-paused iterations will block concurrent writers.
//
// Iteration stops if the consumer abandons the range loop or if the context
// is canceled between yields.
func (es *InMemoryStore) Stream(
	ctx context.Context,
	id StreamID,
	selector version.Selector,
) *Stream {
	return NewStream(func(yield func(Persisted) bool) error {
		es.mx.RLock()
		defer es.mx.RUnlock()

		events, ok := es.events[id]
		if !ok {
			return nil
		}

		for i, evt := range events {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("event.InMemoryStore: context error, %w", err)
			}

			eventVersion := version.Version(i) + 1

			if eventVersion < selector.From {
				continue
			}

			if !yield(Persisted{
				Envelope: evt,
				StreamID: id,
				Version:  eventVersion,
			}) {
				return nil
			}
		}

		return nil
	})
}

// Append inserts the specified Domain Events into the Event Stream specified
// by the current instance, returning the new version of the Event Stream.
//
// `version.CheckExact` can be specified to enable an Optimistic Concurrency check
// on append, by using the expected version of the Event Stream prior
// to appending the new Events.
//
// Alternatively, `version.Any` can be used if no Optimistic Concurrency check
// should be carried out.
//
// An instance of `version.ConflictError` will be returned if the optimistic locking
// version check fails against the current version of the Event Stream.
func (es *InMemoryStore) Append(
	_ context.Context,
	id StreamID,
	expected version.Check,
	events ...Envelope,
) (version.Version, error) {
	es.mx.Lock()
	defer es.mx.Unlock()

	currentVersion := version.CheckExact(len(es.events[id])) //nolint:gosec // This should not overflow.

	if v, expectsExactVersion := expected.(version.CheckExact); expectsExactVersion && currentVersion != expected {
		return 0, fmt.Errorf("event.InMemoryStore: failed to append events, %w", version.ConflictError{
			Expected: version.Version(v),
			Actual:   version.Version(currentVersion),
		})
	}

	es.events[id] = append(es.events[id], events...)
	newEventStreamVersion := version.Version(len(es.events[id])) //nolint:gosec // This should not overflow.

	return newEventStreamVersion, nil
}

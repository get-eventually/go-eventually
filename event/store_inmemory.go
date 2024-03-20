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

func contextErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("event.InMemoryStore: context error, %w", err)
	}

	return nil
}

// Stream streams committed events in the Event Store onto the provided EventStream,
// from the specified Global Sequence Number in `from`, based on the provided stream.Target.
//
// Note: this call is synchronous, and will return when all the Events
// have been successfully written to the provided EventStream, or when
// the context has been canceled.
//
// This method fails only when the context is canceled.
func (es *InMemoryStore) Stream(
	ctx context.Context,
	eventStream StreamWrite,
	id StreamID,
	selector version.Selector,
) error {
	es.mx.RLock()
	defer es.mx.RUnlock()
	defer close(eventStream)

	events, ok := es.events[id]
	if !ok {
		return nil
	}

	for i, evt := range events {
		eventVersion := version.Version(i) + 1

		if eventVersion < selector.From {
			continue
		}

		persistedEvent := Persisted{
			Envelope: evt,
			StreamID: id,
			Version:  eventVersion,
		}

		select {
		case eventStream <- persistedEvent:
		case <-ctx.Done():
			return contextErr(ctx)
		}
	}

	return nil
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

	currentVersion := version.CheckExact(len(es.events[id]))

	if expected != version.Any && currentVersion != expected {
		return 0, fmt.Errorf("event.InMemoryStore: failed to append events, %w", version.ConflictError{
			Expected: version.Version(expected.(version.CheckExact)),
			Actual:   version.Version(currentVersion),
		})
	}

	es.events[id] = append(es.events[id], events...)
	newEventStreamVersion := version.Version(len(es.events[id]))

	return newEventStreamVersion, nil
}

package test

import (
	"context"
	"fmt"
	"sync"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"
)

var _ event.Store = &InMemoryEventStore{}

// InMemoryEventStore is an in-memory event.Store implementation.
type InMemoryEventStore struct {
	mx     sync.RWMutex
	events map[event.StreamID][]event.Envelope
}

// NewInMemoryEventStore creates a new empty test.InMemoryEventStore instance.
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[event.StreamID][]event.Envelope),
	}
}

func contextErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("test.InMemoryEventStore: context error: %w", err)
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
func (es *InMemoryEventStore) Stream(
	ctx context.Context,
	eventStream event.StreamWrite,
	id event.StreamID,
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

		persistedEvent := event.Persisted{
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
// A version can be specified to enable an Optimistic Concurrency check
// on append, by using the expected version of the Event Stream prior
// to appending the new Events.
//
// Alternatively, -1 can be used if no Optimistic Concurrency check
// should be carried out.
//
// An instance of ErrConflict will be returned if the optimistic locking
// version check fails against the current version of the Event Stream.
func (es *InMemoryEventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	es.mx.Lock()
	defer es.mx.Unlock()

	currentVersion := version.CheckExact(len(es.events[id]))

	if expected != version.Any && currentVersion != expected {
		return 0, fmt.Errorf("%T: failed to append events: %w", es, version.ConflictError{
			Expected: version.Version(expected.(version.CheckExact)),
			Actual:   version.Version(currentVersion),
		})
	}

	es.events[id] = append(es.events[id], events...)
	newEventStreamVersion := version.Version(len(es.events[id]))

	return newEventStreamVersion, nil
}

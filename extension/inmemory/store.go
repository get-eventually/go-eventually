package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/event/version"
)

var _ event.Store = &EventStore{}

// EventStore is an in-memory eventstore.Store implementation.
type EventStore struct {
	mx sync.RWMutex

	events            []event.Persisted
	byType            map[string][]int
	byTypeAndInstance map[string]map[string][]int
}

// NewEventStore creates a new empty EventStore instance.
func NewEventStore() *EventStore {
	return &EventStore{
		byType:            make(map[string][]int),
		byTypeAndInstance: make(map[string]map[string][]int),
	}
}

func contextErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("inmemory.EventStore: context error: %w", err)
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
func (s *EventStore) Stream(
	ctx context.Context,
	eventStream event.Stream,
	eventStreamID event.StreamID,
	selector version.Selector,
) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(eventStream)

	if m, ok := s.byTypeAndInstance[eventStreamID.Type]; !ok || m == nil {
		return nil
	}

	eventIdxs, ok := s.byTypeAndInstance[eventStreamID.Type][eventStreamID.Name]
	if !ok {
		return nil
	}

	for _, idx := range eventIdxs {
		evt := s.events[idx]

		if evt.Version < selector.From {
			continue
		}

		select {
		case eventStream <- evt:
		case <-ctx.Done():
			return contextErr(ctx)
		}
	}

	return nil
}

func (s *EventStore) ensureMapsAreCreated(typ string) {
	if v, ok := s.byTypeAndInstance[typ]; !ok || v == nil {
		s.byTypeAndInstance[typ] = make(map[string][]int)
	}
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
func (s *EventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Event,
) (version.Version, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	// Make sure we create the necessary maps, otherwise the code down below will panic.
	s.ensureMapsAreCreated(id.Type)

	currentVersion := version.CheckExact(len(s.byTypeAndInstance[id.Type][id.Name]))
	if expected != version.Any && currentVersion != expected {
		err := version.ConflictError{
			Expected: version.Version(expected.(version.CheckExact)),
			Actual:   version.Version(currentVersion),
		}

		return 0, fmt.Errorf("inmemory.EventStore: failed to append events: %w", err)
	}

	nextOffset := int64(len(s.events))
	newPersistedEvents := make([]event.Persisted, 0, len(events))
	newIndexes := make([]int, 0, len(events))

	for i, evt := range events {
		nextIndex := int(nextOffset) + i

		// Sequence numbers and versions should all start from 1,
		// hence why this block uses `+ 1`.
		newPersistedEvents = append(newPersistedEvents, event.Persisted{
			Stream:         id,
			Version:        version.Version(currentVersion) + version.Version(i) + 1,
			SequenceNumber: version.SequenceNumber(nextIndex) + 1,
			Event:          evt,
		})

		newIndexes = append(newIndexes, nextIndex)
	}

	s.events = append(s.events, newPersistedEvents...)
	s.byType[id.Type] = append(s.byType[id.Type], newIndexes...)
	s.byTypeAndInstance[id.Type][id.Name] = append(s.byTypeAndInstance[id.Type][id.Name], newIndexes...)

	lastCommittedEvent := newPersistedEvents[len(newPersistedEvents)-1]

	return lastCommittedEvent.Version, nil
}

// LatestSequenceNumber returns the size of the internal Event Log.
// This method never fails.
func (s *EventStore) LatestSequenceNumber(ctx context.Context) (int64, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return int64(len(s.events)), nil
}

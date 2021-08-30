package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

var _ eventstore.Store = &EventStore{}

// EventStore is an in-memory eventstore.Store implementation.
type EventStore struct {
	mx sync.RWMutex

	events            []eventstore.Event
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

func (s *EventStore) streamAll(ctx context.Context, es eventstore.EventStream, selectt eventstore.Select) error {
	for _, event := range s.events {
		if event.SequenceNumber < selectt.From {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return contextErr(ctx)
		}
	}

	return nil
}

func (s *EventStore) streamByType(
	ctx context.Context,
	es eventstore.EventStream,
	typ string,
	selectt eventstore.Select,
) error {
	for _, eventIdx := range s.byType[typ] {
		event := s.events[eventIdx]

		if event.SequenceNumber < selectt.From {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return contextErr(ctx)
		}
	}

	return nil
}

func (s *EventStore) streamByTypes(
	ctx context.Context,
	es eventstore.EventStream,
	types []string,
	selectt eventstore.Select,
) error {
	indexedTypes := make(map[string]struct{})
	for _, typ := range types {
		indexedTypes[typ] = struct{}{}
	}

	for _, event := range s.events {
		if event.SequenceNumber < selectt.From {
			continue
		}

		if _, ok := indexedTypes[event.Stream.Type]; !ok {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return contextErr(ctx)
		}
	}

	return nil
}

func (s *EventStore) streamByID(
	ctx context.Context,
	es eventstore.EventStream,
	id stream.ID,
	selectt eventstore.Select,
) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(es)

	if m, ok := s.byTypeAndInstance[id.Type]; !ok || m == nil {
		return nil
	}

	eventIdxs, ok := s.byTypeAndInstance[id.Type][id.Name]
	if !ok {
		return nil
	}

	for _, idx := range eventIdxs {
		event := s.events[idx]

		if event.Version < selectt.From {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return contextErr(ctx)
		}
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
	es eventstore.EventStream,
	target stream.Target,
	selectt eventstore.Select,
) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(es)

	switch t := target.(type) {
	case stream.All:
		return s.streamAll(ctx, es, selectt)
	case stream.ByType:
		return s.streamByType(ctx, es, string(t), selectt)
	case stream.ByTypes:
		return s.streamByTypes(ctx, es, []string(t), selectt)
	case stream.ByID:
		return s.streamByID(ctx, es, stream.ID(t), selectt)
	default:
		return fmt.Errorf("inmemory.EventStore: unsupported stream target type provided: %T", t)
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
	id stream.ID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	// Make sure we create the necessary maps, otherwise the code
	// down below will panic.
	s.ensureMapsAreCreated(id.Type)

	currentVersion := eventstore.VersionCheck(len(s.byTypeAndInstance[id.Type][id.Name]))
	if expected != eventstore.VersionCheckAny && currentVersion != expected {
		return 0, fmt.Errorf("inmemory: failed to append events: %w", eventstore.ErrConflict{
			Expected: int64(expected),
			Actual:   int64(currentVersion),
		})
	}

	nextOffset := int64(len(s.events))
	newPersistedEvents := make([]eventstore.Event, 0, len(events))
	newIndexes := make([]int, 0, len(events))

	for i, event := range events {
		nextIndex := int(nextOffset) + i

		// Sequence numbers and versions should all start from 1,
		// hence why this block uses `+ 1`.
		newPersistedEvents = append(newPersistedEvents, eventstore.Event{
			Stream:         id,
			Version:        int64(currentVersion) + int64(i) + 1,
			SequenceNumber: int64(nextIndex) + 1,
			Event:          event,
		})

		newIndexes = append(newIndexes, nextIndex)
	}

	s.events = append(s.events, newPersistedEvents...)
	s.byType[id.Type] = append(s.byType[id.Type], newIndexes...)
	s.byTypeAndInstance[id.Type][id.Name] = append(s.byTypeAndInstance[id.Type][id.Name], newIndexes...)

	lastCommittedEvent := newPersistedEvents[len(newPersistedEvents)-1]

	return lastCommittedEvent.Version, nil
}

func (s *EventStore) ensureMapsAreCreated(typ string) {
	if v, ok := s.byTypeAndInstance[typ]; !ok || v == nil {
		s.byTypeAndInstance[typ] = make(map[string][]int)
	}
}

func contextErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("inmemory.EventStore: context done: %w", err)
	}

	return nil
}

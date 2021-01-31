package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

var _ eventstore.Store = &EventStore{}

// EventStore is an in-memory eventstore.Store implementation.
type EventStore struct {
	mx sync.RWMutex

	events            []eventstore.Event
	byType            map[string][]int
	byTypeAndInstance map[string]map[string][]int

	subscribers       []eventstore.EventStream
	subscribersByType map[string][]eventstore.EventStream
}

// NewEventStore creates a new empty EventStore instance.
func NewEventStore() *EventStore {
	return &EventStore{
		byType:            make(map[string][]int),
		byTypeAndInstance: make(map[string]map[string][]int),
		subscribersByType: make(map[string][]eventstore.EventStream),
	}
}

// Type returns an eventstore.Typed access instance using the specified
// Stream type identifier.
//
// An error is returned if the type has not been registered yet.
func (s *EventStore) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	_, ok := s.byType[typ]
	if !ok {
		return nil, fmt.Errorf("inmemory.EventStore: type not registered in the event store")
	}

	return typedEventStoreAccess{EventStore: s, typ: typ}, nil
}

// Register registers the provided type identifier.
//
// Note: nil is a valid value for the events mapper, as there is no need
// for this particular Event Store mechanism to register Event types.
//
// This function never fails.
func (s *EventStore) Register(ctx context.Context, typ string, events map[string]interface{}) error {
	if _, ok := s.byType[typ]; !ok {
		s.byType[typ] = nil
	}

	if _, ok := s.subscribersByType[typ]; !ok {
		s.subscribersByType[typ] = nil
	}

	if _, ok := s.byTypeAndInstance[typ]; !ok {
		s.byTypeAndInstance[typ] = make(map[string][]int)
	}

	return nil
}

// Stream streams all Event Store committed events onto the provided EventStream,
// from the specified Global Sequence Number in `from`.
//
// Note: this call is synchronous, and will return when all the Events
// have been successfully written to the provided EventStream, or when
// the context has been canceled.
//
// An error is returned if one Event in the Event Store does not have a
// Global Sequence Number (which should never happen), or when the context
// is done.
func (s *EventStore) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(es)

	for _, event := range s.events {
		sequenceNumber, ok := event.GlobalSequenceNumber()
		if !ok {
			return fmt.Errorf("inmemory.EventStore: event does not have global sequence number")
		}

		if sequenceNumber < from {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return fmt.Errorf("inmemory.EventStore: context done: %w", ctx.Err())
		}
	}

	return nil
}

// Subscribe subscribes to all committed Events in the Event Store and
// uses the provided EventStream to notify the callers with such Events.
//
// Note: this call is synchronous, and returns to the caller
// only when the context is closed.
//
// context.Canceled error is always returned.
func (s *EventStore) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	defer close(es)

	s.mx.Lock()
	s.subscribers = append(s.subscribers, es)
	s.mx.Unlock()

	<-ctx.Done()

	s.mx.Lock()
	defer s.mx.Unlock()

	subscribers := make([]eventstore.EventStream, 0, len(s.subscribers)-1)

	for _, subscriber := range s.subscribers {
		if subscriber == es {
			continue
		}
		subscribers = append(subscribers, subscriber)
	}

	s.subscribers = subscribers

	return fmt.Errorf("inmemory.EventStore: context done: %w", ctx.Err())
}

type typedEventStoreAccess struct {
	*EventStore
	typ string
}

func (s typedEventStoreAccess) Instance(id string) eventstore.Instanced {
	return instanceEventStoreAccess{typedEventStoreAccess: s, id: id}
}

func (s typedEventStoreAccess) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(es)

	for _, eventIdx := range s.byType[s.typ] {
		event := s.events[eventIdx]
		sequenceNumber, ok := event.GlobalSequenceNumber()
		if !ok {
			return fmt.Errorf("inmemory: event does not have global sequence number")
		}

		if sequenceNumber < from {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return fmt.Errorf("inmemory.EventStore: context done: %w", ctx.Err())
		}
	}

	return nil
}

func (s typedEventStoreAccess) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	defer close(es)

	s.mx.Lock()
	s.subscribersByType[s.typ] = append(s.subscribersByType[s.typ], es)
	s.mx.Unlock()

	<-ctx.Done()

	s.mx.Lock()
	defer s.mx.Unlock()

	subscribers := make([]eventstore.EventStream, 0, len(s.subscribersByType[s.typ])-1)

	for _, subscriber := range s.subscribersByType[s.typ] {
		if subscriber == es {
			continue
		}
		subscribers = append(subscribers, subscriber)
	}

	s.subscribersByType[s.typ] = subscribers

	return ctx.Err()
}

type instanceEventStoreAccess struct {
	typedEventStoreAccess
	id string
}

func (s instanceEventStoreAccess) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(es)

	eventIdxs, ok := s.byTypeAndInstance[s.typ][s.id]
	if !ok {
		return nil
	}

	for _, idx := range eventIdxs {
		event := s.events[idx]

		if event.Version < from {
			continue
		}

		select {
		case es <- event:
		case <-ctx.Done():
			return fmt.Errorf("inmemory.EventStore: context done: %w", ctx.Err())
		}
	}

	return nil
}

func (s instanceEventStoreAccess) Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	currentVersion := int64(len(s.byTypeAndInstance[s.typ][s.id]))
	if version != -1 && currentVersion != version {
		return 0, fmt.Errorf("inmemory: invalid version check, expected %d", currentVersion)
	}

	nextOffset := int64(len(s.events))
	newPersistedEvents := make([]eventstore.Event, 0, len(events))
	newIndexes := make([]int, 0, len(events))

	for i, event := range events {
		nextIndex := int(nextOffset) + i

		// Sequence numbers and versions should all start from 1,
		// hence why this block uses `+ 1`.
		newPersistedEvents = append(newPersistedEvents, eventstore.Event{
			StreamType: s.typ,
			StreamName: s.id,
			Version:    currentVersion + int64(i) + 1,
			Event:      event.WithGlobalSequenceNumber(int64(nextIndex) + 1),
		})

		newIndexes = append(newIndexes, nextIndex)
	}

	s.events = append(s.events, newPersistedEvents...)
	s.byType[s.typ] = append(s.byType[s.typ], newIndexes...)
	s.byTypeAndInstance[s.typ][s.id] = append(s.byTypeAndInstance[s.typ][s.id], newIndexes...)

	// Send notifications to subscribers.
	defer func() { s.notify(newPersistedEvents...) }()

	lastCommittedEvent := newPersistedEvents[len(newPersistedEvents)-1]

	return lastCommittedEvent.Version, nil
}

func (s instanceEventStoreAccess) notify(events ...eventstore.Event) {
	subscribers := append(s.subscribers, s.subscribersByType[s.typ]...)

	for _, event := range events {
		for _, subscriber := range subscribers {
			subscriber <- event
		}
	}
}

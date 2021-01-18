package inmemory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

var _ eventstore.Store = &EventStore{}

type EventStore struct {
	mx     sync.RWMutex
	offset int64

	events            []eventstore.Event
	byType            map[string][]eventstore.Event
	byTypeAndInstance map[string]map[string][]eventstore.Event

	subscribers       []eventstore.EventStream
	subscribersByType map[string][]eventstore.EventStream
}

func NewEventStore() *EventStore {
	return &EventStore{
		byType:            make(map[string][]eventstore.Event),
		byTypeAndInstance: make(map[string]map[string][]eventstore.Event),
		subscribersByType: make(map[string][]eventstore.EventStream),
	}
}

func (s *EventStore) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	_, ok := s.byType[typ]
	if !ok {
		return nil, fmt.Errorf("inmemory: type not registered in the event store")
	}

	return typedEventStoreAccess{EventStore: s, typ: typ}, nil
}

func (s *EventStore) Register(ctx context.Context, typ string, events map[string]interface{}) error {
	if _, ok := s.byType[typ]; !ok {
		s.byType[typ] = nil
	}

	if _, ok := s.subscribersByType[typ]; !ok {
		s.subscribersByType[typ] = nil
	}

	if _, ok := s.byTypeAndInstance[typ]; !ok {
		s.byTypeAndInstance[typ] = make(map[string][]eventstore.Event)
	}

	return nil
}

func (s *EventStore) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	s.mx.RLock()
	defer s.mx.RUnlock()
	defer close(es)

	for _, event := range s.events {
		sequenceNumber, ok := event.GlobalSequenceNumber()
		if !ok {
			return fmt.Errorf("inmemory: event does not have global sequence number")
		}

		if sequenceNumber < from {
			continue
		}

		es <- event
	}

	return nil
}

func (s *EventStore) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	defer close(es)

	s.mx.Lock()
	{
		s.subscribers = append(s.subscribers, es)
	}
	s.mx.Unlock()

	<-ctx.Done()

	s.mx.Lock()
	{
		subscribers := make([]eventstore.EventStream, 0, len(s.subscribers)-1)

		for _, subscriber := range s.subscribers {
			if subscriber == es {
				continue
			}
			subscribers = append(subscribers, subscriber)
		}

		s.subscribers = subscribers
	}
	s.mx.Unlock()

	return ctx.Err()
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

	for _, event := range s.byType[s.typ] {
		sequenceNumber, ok := event.GlobalSequenceNumber()
		if !ok {
			return fmt.Errorf("inmemory: event does not have global sequence number")
		}

		if sequenceNumber < from {
			continue
		}

		es <- event
	}

	return nil
}

func (s typedEventStoreAccess) Subscribe(ctx context.Context, es eventstore.EventStream) error {
	defer close(es)

	s.mx.Lock()
	{
		s.subscribersByType[s.typ] = append(s.subscribersByType[s.typ], es)
	}
	s.mx.Unlock()

	<-ctx.Done()

	s.mx.Lock()
	{
		subscribers := make([]eventstore.EventStream, 0, len(s.subscribersByType[s.typ])-1)

		for _, subscriber := range s.subscribersByType[s.typ] {
			if subscriber == es {
				continue
			}
			subscribers = append(subscribers, subscriber)
		}

		s.subscribersByType[s.typ] = subscribers
	}
	s.mx.Unlock()

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

	events, ok := s.byTypeAndInstance[s.typ][s.id]
	if !ok {
		return nil
	}

	for _, event := range events {
		if event.Version < from {
			continue
		}

		es <- event
	}

	return nil
}

func (s instanceEventStoreAccess) Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	persisted, _ := s.byTypeAndInstance[s.typ][s.id]
	currentVersion := int64(len(persisted))

	if version != -1 && currentVersion != version {
		return 0, fmt.Errorf("inmemory: invalid version check, expected %d", currentVersion)
	}

	typedEvents, _ := s.byType[s.typ]

	if len(typedEvents) == 0 {
		typedEvents = make([]eventstore.Event, 0, len(events))
	}

	if currentVersion == 0 {
		persisted = make([]eventstore.Event, 0, len(events))
	}

	for i, event := range events {
		evt := eventstore.Event{
			StreamType: s.typ,
			StreamName: s.id,
			Version:    currentVersion + int64(i) + 1,
			Event:      event.WithGlobalSequenceNumber(atomic.AddInt64(&s.offset, 1)),
		}

		persisted = append(persisted, evt)
		typedEvents = append(typedEvents, evt)
		s.events = append(s.events, evt)
	}

	s.byType[s.typ] = typedEvents
	s.byTypeAndInstance[s.typ][s.id] = persisted

	go func() {
		s.mx.RLock()
		defer s.mx.RUnlock()

		subscribers := append(s.subscribers, s.subscribersByType[s.typ]...)

		for _, event := range persisted {
			for _, subscriber := range subscribers {
				subscriber <- event
			}
		}
	}()

	return int64(len(persisted)), nil
}

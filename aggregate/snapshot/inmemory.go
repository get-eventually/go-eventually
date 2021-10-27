package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/get-eventually/go-eventually/event/version"
)

var (
	_ Recorder = &InMemoryStore{}
	_ Getter   = &InMemoryStore{}
)

// InMemoryStore is a map-based, thread-safe inmemory Snapshot store
// that can be used for storing long-lived Aggregate Roots.
//
// Since there is no entry eviction, it is suggested to use this store
// only for test scenarios.
type InMemoryStore struct {
	mx                     sync.RWMutex
	snapshotsByAggregateID map[string]Snapshot
}

// NewInMemoryStore returns a fresh new instance of an the InMemoryStore snapshot store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		snapshotsByAggregateID: make(map[string]Snapshot),
	}
}

// Record adds or overwrites the previous Aggregate Root state in the store internal state.
// This operation cannot fail.
func (s *InMemoryStore) Record(ctx context.Context, id string, newVersion version.Version, state interface{}) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.snapshotsByAggregateID[id] = Snapshot{
		Version:    newVersion,
		State:      state,
		RecordedAt: time.Now(),
	}

	return nil
}

// Get returns the latest version of the Aggregate Root recorded by its Aggregate id.
// ErrNotFound is returned if no Aggregate Root state has been committed to the store.
func (s *InMemoryStore) Get(ctx context.Context, id string) (Snapshot, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	if snap, ok := s.snapshotsByAggregateID[id]; ok {
		return snap, nil
	}

	return Snapshot{}, ErrNotFound
}

// MarshalJSON serializes the internal state of the store for debugging purposes.
//
// When relying on this functionality, make sure that the fields of your Aggregate Root
// state are correctly exported, or that your Aggregate Root implements json.Unmarshaler
// and json.Marshaler interfaces, for correct (de)-serialization.
func (s *InMemoryStore) MarshalJSON() ([]byte, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	byt, err := json.Marshal(s.snapshotsByAggregateID)
	if err != nil {
		return nil, fmt.Errorf("snapshot.InMemoryStore: failed to marshal internal state to json: %w", err)
	}

	return byt, nil
}

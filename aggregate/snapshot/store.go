package snapshot

import (
	"context"
	"fmt"
)

// ErrNotFound is returned by a snapshot.Getter when no recent snapshot
// has been found in the store.
var ErrNotFound = fmt.Errorf("snapshot: entry not found")

// Snapshot represents the value of a snapshot found in the store.
type Snapshot struct {
	Version int64       `json:"version"`
	State   interface{} `json:"state"`
}

// Recorder is used to record Snapshots to a durable store.
type Recorder interface {
	Record(ctx context.Context, id string, version int64, state interface{}) error
}

// Getter is used to retrieve the most-recent Snapshot from a durable store.
type Getter interface {
	Get(ctx context.Context, id string) (Snapshot, error)
}

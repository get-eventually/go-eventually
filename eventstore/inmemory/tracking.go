package inmemory

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

type TrackingEventStore struct {
	eventstore.Typed
	recorded []eventually.Event
}

func (es *TrackingEventStore) Recorded() []eventually.Event { return es.recorded }

func (es *TrackingEventStore) Instance(id string) eventstore.Instanced {
	return &instancedTrackingEventStore{
		parent:    es,
		Instanced: es.Typed.Instance(id),
	}
}

type instancedTrackingEventStore struct {
	eventstore.Instanced
	parent *TrackingEventStore
}

func (es *instancedTrackingEventStore) Append(
	ctx context.Context,
	version int64,
	events ...eventually.Event,
) (int64, error) {
	v, err := es.Instanced.Append(ctx, version, events...)

	if err == nil {
		es.parent.recorded = append(es.parent.recorded, events...)
	}

	return v, err
}

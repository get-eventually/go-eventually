package event_test

import (
	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/extension/correlation"
	"github.com/get-eventually/go-eventually/extension/inmemory"
)

func ExampleFusedStore() {
	eventStore := inmemory.NewEventStore()
	correlatedEventStore := correlation.EventStoreWrapper{
		Appender:  eventStore,
		Generator: func() string { return "test-id" },
	}

	aggregate.NewRepository(aggregate.Type{}, event.FusedStore{
		Appender: correlatedEventStore,
		Streamer: eventStore,
	})
}

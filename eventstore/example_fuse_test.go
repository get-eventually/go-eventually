package eventstore_test

import (
	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/extension/correlation"
)

func ExampleFused() {
	eventStore := inmemory.NewEventStore()
	correlatedEventStore := correlation.EventStoreWrapper{
		Appender:  eventStore,
		Generator: func() string { return "test-id" },
	}

	aggregate.NewRepository(aggregate.Type{}, eventstore.Fused{
		Appender: correlatedEventStore,
		Streamer: eventStore,
	})
}

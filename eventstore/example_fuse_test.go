package eventstore_test

import (
	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/extension/correlation"
)

func ExampleFusedAppendStreamer() {
	eventStore := inmemory.NewEventStore()
	correlatedEventStore := correlation.WrapEventStore(eventStore, func() string {
		return "test-id"
	})

	aggregate.NewRepository(aggregate.Type{}, eventstore.FusedAppendStreamer{
		Appender: correlatedEventStore,
		Streamer: eventStore,
	})
}

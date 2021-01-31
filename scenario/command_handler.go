package scenario

import (
	"context"
	"errors"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/aggregate"
	"github.com/eventually-rs/eventually-go/command"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"

	"github.com/stretchr/testify/assert"
)

type CommandHandlerInit struct{}

func CommandHandler() CommandHandlerInit { return CommandHandlerInit{} }

func (sc CommandHandlerInit) Given(events ...eventstore.Event) CommandHandlerGiven {
	return CommandHandlerGiven{given: events}
}

func (sc CommandHandlerInit) When(command eventually.Command) CommandHandlerWhen {
	return CommandHandlerWhen{when: command}
}

type CommandHandlerGiven struct {
	given []eventstore.Event
}

func (sc CommandHandlerGiven) When(command eventually.Command) CommandHandlerWhen {
	return CommandHandlerWhen{
		CommandHandlerGiven: sc,
		when:                command,
	}
}

type CommandHandlerWhen struct {
	CommandHandlerGiven
	when eventually.Command
}

func (sc CommandHandlerWhen) Then(events ...eventstore.Event) CommandHandlerThen {
	return CommandHandlerThen{
		CommandHandlerWhen: sc,
		then:               events,
	}
}

func (sc CommandHandlerWhen) ThenError(err error) CommandHandlerThen {
	return CommandHandlerThen{
		CommandHandlerWhen: sc,
		wantError:          true,
		thenError:          err,
	}
}

func (sc CommandHandlerWhen) ThenFails() CommandHandlerThen {
	return CommandHandlerThen{
		CommandHandlerWhen: sc,
		wantError:          true,
	}
}

type CommandHandlerThen struct {
	CommandHandlerWhen
	then      []eventstore.Event
	thenError error
	wantError bool
}

func (sc CommandHandlerThen) Using(
	t *testing.T,
	aggregateType aggregate.Type,
	handlerFactory func(*aggregate.Repository) command.Handler,
) {
	ctx := context.Background()
	store := inmemory.NewEventStore()

	for _, event := range sc.given {
		if err := store.Register(ctx, event.StreamType, nil); !assert.NoError(t, err) {
			return
		}

		typedStore, err := store.Type(ctx, event.StreamType)
		if !assert.NoError(t, err) {
			return
		}

		_, err = typedStore.
			Instance(event.StreamName).
			Append(context.Background(), -1, event.Event)

		if !assert.NoError(t, err) {
			return
		}
	}

	// Register the target aggregate type.
	if err := store.Register(ctx, aggregateType.Name(), nil); !assert.NoError(t, err) {
		return
	}

	trackingStore := &inmemory.TrackingEventStore{Store: store}
	typedStore, err := trackingStore.Type(ctx, aggregateType.Name())

	if !assert.NoError(t, err) {
		return
	}

	repository := aggregate.NewRepository(aggregateType, typedStore)

	handler := handlerFactory(repository)
	err = handler.Handle(context.Background(), sc.when)

	if !sc.wantError {
		assert.NoError(t, err)
		assert.Equal(t, sc.then, trackingStore.Recorded())

		return
	}

	if !assert.Error(t, err) {
		return
	}

	if sc.thenError != nil && !assert.True(t, errors.Is(err, sc.thenError)) {
		t.Log("Unexpected error received:", err)
		return
	}
}

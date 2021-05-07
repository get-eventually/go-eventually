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

func (sc CommandHandlerInit) When(cmd eventually.Command) CommandHandlerWhen {
	return CommandHandlerWhen{when: cmd}
}

type CommandHandlerGiven struct {
	given []eventstore.Event
}

func (sc CommandHandlerGiven) When(cmd eventually.Command) CommandHandlerWhen {
	return CommandHandlerWhen{
		CommandHandlerGiven: sc,
		when:                cmd,
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

func (sc CommandHandlerThen) Using( //nolint:gocritic
	t *testing.T,
	aggregateType aggregate.Type,
	handlerFactory func(*aggregate.Repository) command.Handler,
) {
	ctx := context.Background()
	store := inmemory.NewEventStore()

	for _, event := range sc.given {
		_, err := store.Append(ctx, event.StreamID, eventstore.VersionCheckAny, event.Event)
		if !assert.NoError(t, err) {
			return
		}
	}

	trackingStore := inmemory.NewTrackingEventStore(store)
	repository := aggregate.NewRepository(aggregateType, trackingStore)

	handler := handlerFactory(repository)
	err := handler.Handle(context.Background(), sc.when)

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

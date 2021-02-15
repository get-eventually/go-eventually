package scenario

import (
	"context"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/command"
	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/projection"
	"github.com/eventually-rs/eventually-go/query"
)

type ProcessManagerInit struct{}

func ProcessManager() ProcessManagerInit { return ProcessManagerInit{} }

func (ProcessManagerInit) Given(events ...eventstore.Event) ProcessManagerGiven {
	return ProcessManagerGiven{
		given: events,
	}
}

type ProcessManagerGiven struct {
	given []eventstore.Event
}

func (s ProcessManagerGiven) When(event eventstore.Event) ProcessManagerWhen {
	return ProcessManagerWhen{
		ProcessManagerGiven: s,
		when:                event,
	}
}

type ProcessManagerWhen struct {
	ProcessManagerGiven
	when eventstore.Event
}

func (s ProcessManagerWhen) Then(command eventually.Command) ProcessManagerThen {
	return ProcessManagerThen{
		ProcessManagerWhen: s,
		then:               command,
	}
}

type QueryFactory func() query.Handler

type ProcessManagerFactory func(query.Dispatcher, command.Dispatcher) projection.Applier

type commandDispatcherFunc func(context.Context, eventually.Command) error

func (fn commandDispatcherFunc) Dispatch(ctx context.Context, cmd eventually.Command) error {
	return fn(ctx, cmd)
}

type ProcessManagerThen struct {
	ProcessManagerWhen
	then eventually.Command
}

func (s ProcessManagerThen) Using(
	t *testing.T,
	queryFactory QueryFactory,
	processManagerFactory ProcessManagerFactory,
) {
	// queryBus := query.NewSimpleBus()
	// queryBus.Register(queryFactory())

	// commandDispatcher := commandDispatcherFunc(func(ctx context.Context, cmd eventually.Command) error {
	// 	assert.Equal(t, s.then, cmd)
	// 	return nil
	// })

	// processManager := processManagerFactory(queryBus, commandDispatcher)

	// assert.NoError(t, processManager.Apply(context.Background(), s.))
	panic("implement me")
}

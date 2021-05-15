package scenario

import (
	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
)

type ProcessManagerInit struct{}

func ProcessManager() ProcessManagerInit {
	return ProcessManagerInit{}
}

func (ProcessManagerInit) Given(events ...eventstore.Event) ProcessManagerGiven {
	return ProcessManagerGiven{
		given: events,
	}
}

type ProcessManagerGiven struct {
	given []eventstore.Event
}

func (sc ProcessManagerGiven) When(event eventstore.Event) ProcessManagerWhen {
	return ProcessManagerWhen{
		ProcessManagerGiven: sc,
		when:                event,
	}
}

type ProcessManagerWhen struct {
	ProcessManagerGiven

	when eventstore.Event
}

func (sc ProcessManagerWhen) ThenCommands(commands ...eventually.Command) ProcessManagerThen {
	return ProcessManagerThen{
		ProcessManagerWhen: sc,
		thenCommands:       commands,
	}
}

func (sc ProcessManagerWhen) ThenEvents(events ...eventually.Event) ProcessManagerThen {
	return ProcessManagerThen{
		ProcessManagerWhen: sc,
		thenEvents:         events,
	}
}

type ProcessManagerThen struct {
	ProcessManagerWhen

	thenCommands []eventually.Command
	thenEvents   []eventually.Event
}

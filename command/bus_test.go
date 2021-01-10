package command_test

import (
	"context"
	"testing"

	"github.com/eventually-rs/eventually-go/command"

	"github.com/stretchr/testify/assert"
)

type cmd struct {
	id string
}

func (cmd cmd) AggregateID() string { return cmd.id }

type handler struct {
	t *testing.T
}

func (handler) CommandType() command.Command { return cmd{} }
func (h handler) Handle(ctx context.Context, c command.Command) error {
	assert.IsType(h.t, cmd{}, c)
	return nil
}

func TestBus(t *testing.T) {
	bus := command.NewSimpleBus()
	bus.Register(handler{t})

	err := bus.Dispatch(context.Background(), cmd{id: "test"})
	assert.NoError(t, err)
}

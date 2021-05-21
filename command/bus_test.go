package command_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/command"
)

type cmd struct{}

func (cmd) Name() string { return "cmd" }

type handler struct {
	t *testing.T
}

func (handler) CommandType() command.Command { return cmd{} }
func (h handler) Handle(ctx context.Context, c eventually.Command) error {
	assert.IsType(h.t, cmd{}, c.Payload)
	return nil
}

func TestBus(t *testing.T) {
	bus := command.NewSimpleBus()
	bus.Register(handler{t})

	err := bus.Dispatch(context.Background(), eventually.Command{
		Payload: cmd{},
	})
	assert.NoError(t, err)
}

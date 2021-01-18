package command_test

import (
	"context"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/command"

	"github.com/stretchr/testify/assert"
)

type cmd struct{}

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

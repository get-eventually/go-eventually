package command_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/command"
)

type cmd struct{}

func (cmd) Name() string { return "cmd" }

type handler struct {
	t *testing.T
}

func (handler) CommandType() command.Type { return cmd{} }
func (h handler) Handle(ctx context.Context, c command.Command) error {
	assert.IsType(h.t, cmd{}, c.Payload)
	return nil
}

func TestDispatcher(t *testing.T) {
	dispatcher := command.NewInMemoryDispatcher()
	dispatcher.Register(handler{t})

	err := dispatcher.Dispatch(context.Background(), command.Command{
		Payload: cmd{},
	})
	assert.NoError(t, err)
}

package todolist

import (
	"context"
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/command"
)

// CreateCommand is the Command used to create a new TodoList.
type CreateCommand struct {
	ID    ID
	Title string
	Owner string
}

// Name implements message.Message.
func (CreateCommand) Name() string { return "CreateTodoList" }

//nolint:exhaustruct // Interface implementation assertion.
var _ command.Handler[CreateCommand] = CreateCommandHandler{}

// CreateCommandHandler is the Command Handler for CreateCommand commands.
type CreateCommandHandler struct {
	Clock      func() time.Time
	Repository Saver
}

// Handle implements command.Handler.
func (h CreateCommandHandler) Handle(ctx context.Context, cmd command.Envelope[CreateCommand]) error {
	now := h.Clock()

	tl, err := Create(cmd.Message.ID, cmd.Message.Title, cmd.Message.Owner, now)
	if err != nil {
		return fmt.Errorf("todolist.CreateCommandHandler: failed to create new todolist, %w", err)
	}

	if err := h.Repository.Save(ctx, tl); err != nil {
		return fmt.Errorf("todolist.CreateCommandHandler: failed to save todolist to repository, %w", err)
	}

	return nil
}

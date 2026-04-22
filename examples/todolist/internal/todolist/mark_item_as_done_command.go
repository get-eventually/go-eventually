package todolist

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/command"
)

// MarkItemAsDoneCommand is the Command used to mark an Item in a TodoList
// as completed.
type MarkItemAsDoneCommand struct {
	TodoListID ID
	TodoItemID ItemID
}

// Name implements message.Message.
func (MarkItemAsDoneCommand) Name() string { return "MarkTodoListItemAsDone" }

//nolint:exhaustruct // Interface implementation assertion.
var _ command.Handler[MarkItemAsDoneCommand] = MarkItemAsDoneCommandHandler{}

// MarkItemAsDoneCommandHandler is the command.Handler for
// MarkItemAsDoneCommand commands.
type MarkItemAsDoneCommandHandler struct {
	Repository Repository
}

// Handle implements command.Handler.
func (h MarkItemAsDoneCommandHandler) Handle(
	ctx context.Context,
	cmd command.Envelope[MarkItemAsDoneCommand],
) error {
	tl, err := h.Repository.Get(ctx, cmd.Message.TodoListID)
	if err != nil {
		return fmt.Errorf("todolist.MarkItemAsDoneCommandHandler: failed to get TodoList from repository, %w", err)
	}

	if err := tl.MarkItemAsDone(cmd.Message.TodoItemID); err != nil {
		return fmt.Errorf("todolist.MarkItemAsDoneCommandHandler: failed to mark item as done, %w", err)
	}

	if err := h.Repository.Save(ctx, tl); err != nil {
		return fmt.Errorf("todolist.MarkItemAsDoneCommandHandler: failed to save new TodoList version, %w", err)
	}

	return nil
}

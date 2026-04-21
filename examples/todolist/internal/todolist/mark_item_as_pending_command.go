package todolist

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/command"
)

// MarkItemAsPendingCommand is the Command used to mark an Item in a
// TodoList as pending (i.e. undoing a previous "mark as done").
type MarkItemAsPendingCommand struct {
	TodoListID ID
	TodoItemID ItemID
}

// Name implements message.Message.
func (MarkItemAsPendingCommand) Name() string { return "MarkTodoListItemAsPending" }

//nolint:exhaustruct // Interface implementation assertion.
var _ command.Handler[MarkItemAsPendingCommand] = MarkItemAsPendingCommandHandler{}

// MarkItemAsPendingCommandHandler is the command.Handler for
// MarkItemAsPendingCommand commands.
type MarkItemAsPendingCommandHandler struct {
	Repository Repository
}

// Handle implements command.Handler.
func (h MarkItemAsPendingCommandHandler) Handle(
	ctx context.Context,
	cmd command.Envelope[MarkItemAsPendingCommand],
) error {
	tl, err := h.Repository.Get(ctx, cmd.Message.TodoListID)
	if err != nil {
		return fmt.Errorf("todolist.MarkItemAsPendingCommandHandler: failed to get TodoList from repository, %w", err)
	}

	if err := tl.MarkItemAsPending(cmd.Message.TodoItemID); err != nil {
		return fmt.Errorf("todolist.MarkItemAsPendingCommandHandler: failed to mark item as pending, %w", err)
	}

	if err := h.Repository.Save(ctx, tl); err != nil {
		return fmt.Errorf("todolist.MarkItemAsPendingCommandHandler: failed to save new TodoList version, %w", err)
	}

	return nil
}

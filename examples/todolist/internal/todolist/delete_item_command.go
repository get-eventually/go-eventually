package todolist

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/command"
)

// DeleteItemCommand is the Command used to remove an Item from a TodoList.
type DeleteItemCommand struct {
	TodoListID ID
	TodoItemID ItemID
}

// Name implements message.Message.
func (DeleteItemCommand) Name() string { return "DeleteTodoListItem" }

//nolint:exhaustruct // Interface implementation assertion.
var _ command.Handler[DeleteItemCommand] = DeleteItemCommandHandler{}

// DeleteItemCommandHandler is the command.Handler for DeleteItemCommand
// commands.
type DeleteItemCommandHandler struct {
	Repository Repository
}

// Handle implements command.Handler.
func (h DeleteItemCommandHandler) Handle(
	ctx context.Context,
	cmd command.Envelope[DeleteItemCommand],
) error {
	tl, err := h.Repository.Get(ctx, cmd.Message.TodoListID)
	if err != nil {
		return fmt.Errorf("todolist.DeleteItemCommandHandler: failed to get TodoList from repository, %w", err)
	}

	if err := tl.DeleteItem(cmd.Message.TodoItemID); err != nil {
		return fmt.Errorf("todolist.DeleteItemCommandHandler: failed to delete item, %w", err)
	}

	if err := h.Repository.Save(ctx, tl); err != nil {
		return fmt.Errorf("todolist.DeleteItemCommandHandler: failed to save new TodoList version, %w", err)
	}

	return nil
}

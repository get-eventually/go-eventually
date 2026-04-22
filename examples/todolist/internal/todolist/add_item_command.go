package todolist

import (
	"context"
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/command"
)

// AddItemCommand is the Command used to add a new Item to an existing TodoList.
type AddItemCommand struct {
	TodoListID  ID
	TodoItemID  ItemID
	Title       string
	Description string
	DueDate     time.Time
}

// Name implements message.Message.
func (AddItemCommand) Name() string { return "AddTodoListItem" }

//nolint:exhaustruct // Interface implementation assertion.
var _ command.Handler[AddItemCommand] = AddItemCommandHandler{}

// AddItemCommandHandler is the command.Handler for AddItemCommand commands.
type AddItemCommandHandler struct {
	Clock      func() time.Time
	Repository Repository
}

// Handle implements command.Handler.
func (h AddItemCommandHandler) Handle(ctx context.Context, cmd command.Envelope[AddItemCommand]) error {
	tl, err := h.Repository.Get(ctx, cmd.Message.TodoListID)
	if err != nil {
		return fmt.Errorf("todolist.AddItemCommandHandler: failed to get TodoList from repository, %w", err)
	}

	now := h.Clock()

	if err := tl.AddItem(
		cmd.Message.TodoItemID,
		cmd.Message.Title,
		cmd.Message.Description,
		cmd.Message.DueDate,
		now,
	); err != nil {
		return fmt.Errorf("todolist.AddItemCommandHandler: failed to add item to TodoList, %w", err)
	}

	if err := h.Repository.Save(ctx, tl); err != nil {
		return fmt.Errorf("todolist.AddItemCommandHandler: failed to save new TodoList version, %w", err)
	}

	return nil
}

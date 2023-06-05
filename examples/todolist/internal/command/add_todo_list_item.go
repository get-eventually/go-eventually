package command

import (
	"context"
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
)

// AddTodoListItem the Command used to add a new Item to an existing TodoList.
type AddTodoListItem struct {
	TodoListID  todolist.ID
	TodoItemID  todolist.ItemID
	Title       string
	Description string
	DueDate     time.Time
}

// Name implements message.Message.
func (AddTodoListItem) Name() string { return "AddTodoListItem" }

var _ command.Handler[AddTodoListItem] = AddTodoListItemHandler{}

// AddTodoListItemHandler is the command.Handler for AddTodoListItem commands.
type AddTodoListItemHandler struct {
	Clock      func() time.Time
	Repository todolist.Repository
}

// Handle implements command.Handler.
func (h AddTodoListItemHandler) Handle(ctx context.Context, cmd command.Envelope[AddTodoListItem]) error {
	todoList, err := h.Repository.Get(ctx, cmd.Message.TodoListID)
	if err != nil {
		return fmt.Errorf("command.AddTodoListItem: failed to get TodoList from repository, %w", err)
	}

	now := h.Clock()

	if err := todoList.AddItem(
		cmd.Message.TodoItemID,
		cmd.Message.Title,
		cmd.Message.Description,
		cmd.Message.DueDate,
		now,
	); err != nil {
		return fmt.Errorf("command.AddTodoListItem: failed to add item to TodoList, %w", err)
	}

	if err := h.Repository.Save(ctx, todoList); err != nil {
		return fmt.Errorf("command.AddTodoListItem: failed to save new TodoList version, %w", err)
	}

	return nil
}

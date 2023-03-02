package command

import (
	"context"
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/examples/todolist/domain/todolist"
)

type CreateTodoList struct {
	ID    todolist.ID
	Title string
	Owner string
}

func (CreateTodoList) Name() string { return "CreateTodoList" }

var _ command.Handler[CreateTodoList] = CreateTodoListHandler{}

type CreateTodoListHandler struct {
	Clock      func() time.Time
	Repository todolist.Saver
}

// Handle implements command.Handler
func (h CreateTodoListHandler) Handle(ctx context.Context, cmd command.Envelope[CreateTodoList]) error {
	now := h.Clock()

	todoList, err := todolist.Create(cmd.Message.ID, cmd.Message.Title, cmd.Message.Owner, now)
	if err != nil {
		return fmt.Errorf("command.CreateTodoListHandler: failed to create new todolist, %w", err)
	}

	if err := h.Repository.Save(ctx, todoList); err != nil {
		return fmt.Errorf("command.CreateTodoListHandler: failed to save todolist to repository, %w", err)
	}

	return nil
}

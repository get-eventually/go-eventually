package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
	"github.com/get-eventually/go-eventually/query"
)

// GetTodoList is a Domain Query used to return a TodoList view.
type GetTodoList struct {
	ID todolist.ID
}

// Name implements message.Message.
func (GetTodoList) Name() string { return "GetTodoList" }

var _ query.Handler[GetTodoList, *todolist.TodoList] = GetTodoListHandler{}

// GetTodoListHandler handles a GetTodoList query, returning the TodoList
// aggregate root specified by the query.
type GetTodoListHandler struct {
	Getter todolist.Getter
}

// Handle implements query.Handler.
func (h GetTodoListHandler) Handle(
	ctx context.Context,
	q query.Envelope[GetTodoList],
) (*todolist.TodoList, error) {
	if q.Message.ID == todolist.ID(uuid.Nil) {
		return nil, fmt.Errorf("query.GetTodoList: invalid query provided, %w", todolist.ErrEmptyID)
	}

	tl, err := h.Getter.Get(ctx, q.Message.ID)
	if err != nil {
		return nil, fmt.Errorf("query.GetTodoList: failed to get TodoList from repository, %w", err)
	}

	return tl, nil
}

package todolist

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/query"
)

// GetQuery is the Domain Query used to return a TodoList view.
type GetQuery struct {
	ID ID
}

// Name implements message.Message.
func (GetQuery) Name() string { return "GetTodoList" }

//nolint:exhaustruct // Interface implementation assertion.
var _ query.Handler[GetQuery, *TodoList] = GetQueryHandler{}

// GetQueryHandler handles a GetQuery by returning the TodoList specified
// by the query's ID.
type GetQueryHandler struct {
	Getter Getter
}

// Handle implements query.Handler.
func (h GetQueryHandler) Handle(
	ctx context.Context,
	q query.Envelope[GetQuery],
) (*TodoList, error) {
	if q.Message.ID == ID(uuid.Nil) {
		return nil, fmt.Errorf("todolist.GetQueryHandler: invalid query provided, %w", ErrEmptyID)
	}

	tl, err := h.Getter.Get(ctx, q.Message.ID)
	if err != nil {
		return nil, fmt.Errorf("todolist.GetQueryHandler: failed to get TodoList from repository, %w", err)
	}

	return tl, nil
}

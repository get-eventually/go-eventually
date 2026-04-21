package todolist

import (
	"context"
	"errors"
	"fmt"
	"time"

	connect "connectrpc.com/connect"
	"github.com/google/uuid"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/command"
	todolistv1 "github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1"
	"github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1/todolistv1connect"
	"github.com/get-eventually/go-eventually/query"
)

var _ todolistv1connect.TodoListServiceHandler = (*ConnectServiceHandler)(nil)

// ConnectServiceHandler is the Connect transport for the TodoList service.
//
// Clients generate IDs for new resources and pass them in the request; the
// server responds to commands with google.protobuf.Empty. This keeps
// commands idempotent and free of response-payload coupling.
type ConnectServiceHandler struct {
	todolistv1connect.UnimplementedTodoListServiceHandler

	GetQueryHandler                 GetQueryHandler
	CreateCommandHandler            CreateCommandHandler
	AddItemCommandHandler           AddItemCommandHandler
	MarkItemAsDoneCommandHandler    MarkItemAsDoneCommandHandler
	MarkItemAsPendingCommandHandler MarkItemAsPendingCommandHandler
	DeleteItemCommandHandler        DeleteItemCommandHandler
}

// parseUUIDField converts a string field into a uuid.UUID, returning an
// InvalidArgument Connect error on failure.
func parseUUIDField(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("todolist.ConnectServiceHandler: failed to parse %s as uuid, %w", field, err),
		)
	}

	return id, nil
}

// mapCommandError classifies command-handler errors into Connect codes.
//
// The error is included verbatim (via %w) so clients can see the full chain.
// This is example-appropriate; production code would typically surface only
// a stable, sanitized message per code.
func mapCommandError(op string, err error) *connect.Error {
	code := connect.CodeInternal

	switch {
	case errors.Is(err, ErrEmptyID),
		errors.Is(err, ErrEmptyTitle),
		errors.Is(err, ErrNoOwnerSpecified),
		errors.Is(err, ErrEmptyItemID),
		errors.Is(err, ErrEmptyItemTitle):
		code = connect.CodeInvalidArgument

	case errors.Is(err, ErrItemAlreadyExists):
		code = connect.CodeAlreadyExists

	case errors.Is(err, ErrItemNotFound),
		errors.Is(err, aggregate.ErrRootNotFound):
		code = connect.CodeNotFound
	}

	return connect.NewError(code, fmt.Errorf("%s: %w", op, err))
}

// CreateTodoList implements the Connect service handler.
func (srv *ConnectServiceHandler) CreateTodoList(
	ctx context.Context,
	req *connect.Request[todolistv1.CreateTodoListRequest],
) (*connect.Response[emptypb.Empty], error) {
	id, err := parseUUIDField("todo_list_id", req.Msg.TodoListId)
	if err != nil {
		return nil, err
	}

	cmd := command.ToEnvelope(CreateCommand{
		ID:    ID(id),
		Title: req.Msg.Title,
		Owner: req.Msg.Owner,
	})

	if err := srv.CreateCommandHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("todolist.ConnectServiceHandler.CreateTodoList", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetTodoList implements the Connect service handler.
func (srv *ConnectServiceHandler) GetTodoList(
	ctx context.Context,
	req *connect.Request[todolistv1.GetTodoListRequest],
) (*connect.Response[todolistv1.GetTodoListResponse], error) {
	id, err := parseUUIDField("todo_list_id", req.Msg.TodoListId)
	if err != nil {
		return nil, err
	}

	q := query.ToEnvelope(GetQuery{ID: ID(id)})

	tl, err := srv.GetQueryHandler.Handle(ctx, q)
	if err != nil {
		return nil, mapCommandError("todolist.ConnectServiceHandler.GetTodoList", err)
	}

	return connect.NewResponse(&todolistv1.GetTodoListResponse{
		TodoList: ToProto(tl),
	}), nil
}

// AddTodoItem implements the Connect service handler.
func (srv *ConnectServiceHandler) AddTodoItem(
	ctx context.Context,
	req *connect.Request[todolistv1.AddTodoItemRequest],
) (*connect.Response[emptypb.Empty], error) {
	listID, err := parseUUIDField("todo_list_id", req.Msg.TodoListId)
	if err != nil {
		return nil, err
	}

	itemID, err := parseUUIDField("todo_item_id", req.Msg.TodoItemId)
	if err != nil {
		return nil, err
	}

	var dueDate time.Time
	if req.Msg.DueDate != nil {
		dueDate = req.Msg.DueDate.AsTime()
	}

	cmd := command.ToEnvelope(AddItemCommand{
		TodoListID:  ID(listID),
		TodoItemID:  ItemID(itemID),
		Title:       req.Msg.Title,
		Description: req.Msg.Description,
		DueDate:     dueDate,
	})

	if err := srv.AddItemCommandHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("todolist.ConnectServiceHandler.AddTodoItem", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// parseListAndItemIDs extracts and validates both UUID identifiers that
// appear in every per-item request.
func parseListAndItemIDs(todoListID, todoItemID string) (ID, ItemID, error) {
	listID, err := parseUUIDField("todo_list_id", todoListID)
	if err != nil {
		return ID(uuid.Nil), ItemID(uuid.Nil), err
	}

	itemID, err := parseUUIDField("todo_item_id", todoItemID)
	if err != nil {
		return ID(uuid.Nil), ItemID(uuid.Nil), err
	}

	return ID(listID), ItemID(itemID), nil
}

// MarkTodoItemAsDone implements the Connect service handler.
func (srv *ConnectServiceHandler) MarkTodoItemAsDone(
	ctx context.Context,
	req *connect.Request[todolistv1.MarkTodoItemAsDoneRequest],
) (*connect.Response[emptypb.Empty], error) {
	listID, itemID, err := parseListAndItemIDs(req.Msg.TodoListId, req.Msg.TodoItemId)
	if err != nil {
		return nil, err
	}

	cmd := command.ToEnvelope(MarkItemAsDoneCommand{
		TodoListID: listID,
		TodoItemID: itemID,
	})

	if err := srv.MarkItemAsDoneCommandHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("todolist.ConnectServiceHandler.MarkTodoItemAsDone", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// MarkTodoItemAsPending implements the Connect service handler.
func (srv *ConnectServiceHandler) MarkTodoItemAsPending(
	ctx context.Context,
	req *connect.Request[todolistv1.MarkTodoItemAsPendingRequest],
) (*connect.Response[emptypb.Empty], error) {
	listID, itemID, err := parseListAndItemIDs(req.Msg.TodoListId, req.Msg.TodoItemId)
	if err != nil {
		return nil, err
	}

	cmd := command.ToEnvelope(MarkItemAsPendingCommand{
		TodoListID: listID,
		TodoItemID: itemID,
	})

	if err := srv.MarkItemAsPendingCommandHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("todolist.ConnectServiceHandler.MarkTodoItemAsPending", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// DeleteTodoItem implements the Connect service handler.
func (srv *ConnectServiceHandler) DeleteTodoItem(
	ctx context.Context,
	req *connect.Request[todolistv1.DeleteTodoItemRequest],
) (*connect.Response[emptypb.Empty], error) {
	listID, itemID, err := parseListAndItemIDs(req.Msg.TodoListId, req.Msg.TodoItemId)
	if err != nil {
		return nil, err
	}

	cmd := command.ToEnvelope(DeleteItemCommand{
		TodoListID: listID,
		TodoItemID: itemID,
	})

	if err := srv.DeleteItemCommandHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("todolist.ConnectServiceHandler.DeleteTodoItem", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

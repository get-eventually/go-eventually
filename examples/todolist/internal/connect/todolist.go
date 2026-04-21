// Package connect contains the Connect server implementation for the TodoList
// service.
//
// This package deliberately uses the import alias "connect" for
// connectrpc.com/connect to match the framework's own naming; the package
// name is kept short because it exclusively hosts the Connect transport.
package connect

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
	appcommand "github.com/get-eventually/go-eventually/examples/todolist/internal/command"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/protoconv"
	appquery "github.com/get-eventually/go-eventually/examples/todolist/internal/query"
	"github.com/get-eventually/go-eventually/query"
)

//nolint:exhaustruct // Interface implementation assertion.
var _ todolistv1connect.TodoListServiceHandler = TodoListServiceServer{}

// TodoListServiceServer is the Connect server implementation for the TodoList
// service.
//
// Clients generate IDs for new resources and pass them in the request; the
// server responds to commands with google.protobuf.Empty. This keeps
// commands idempotent and free of response-payload coupling.
type TodoListServiceServer struct {
	todolistv1connect.UnimplementedTodoListServiceHandler

	GetTodoListHandler    appquery.GetTodoListHandler
	CreateTodoListHandler appcommand.CreateTodoListHandler
	AddTodoListHandler    appcommand.AddTodoListItemHandler
}

// parseUUID converts a string field into a uuid.UUID, returning an
// InvalidArgument Connect error on failure.
func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("connect: failed to parse %s as uuid, %w", field, err),
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
	case errors.Is(err, todolist.ErrEmptyID),
		errors.Is(err, todolist.ErrEmptyTitle),
		errors.Is(err, todolist.ErrNoOwnerSpecified),
		errors.Is(err, todolist.ErrEmptyItemID),
		errors.Is(err, todolist.ErrEmptyItemTitle):
		code = connect.CodeInvalidArgument

	case errors.Is(err, todolist.ErrItemAlreadyExists):
		code = connect.CodeAlreadyExists

	case errors.Is(err, todolist.ErrItemNotFound),
		errors.Is(err, aggregate.ErrRootNotFound):
		code = connect.CodeNotFound
	}

	return connect.NewError(code, fmt.Errorf("%s: %w", op, err))
}

// CreateTodoList implements the Connect service handler.
func (srv TodoListServiceServer) CreateTodoList(
	ctx context.Context,
	req *connect.Request[todolistv1.CreateTodoListRequest],
) (*connect.Response[emptypb.Empty], error) {
	id, err := parseUUID("todo_list_id", req.Msg.TodoListId)
	if err != nil {
		return nil, err
	}

	cmd := command.ToEnvelope(appcommand.CreateTodoList{
		ID:    todolist.ID(id),
		Title: req.Msg.Title,
		Owner: req.Msg.Owner,
	})

	if err := srv.CreateTodoListHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("connect.CreateTodoList", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetTodoList implements the Connect service handler.
func (srv TodoListServiceServer) GetTodoList(
	ctx context.Context,
	req *connect.Request[todolistv1.GetTodoListRequest],
) (*connect.Response[todolistv1.GetTodoListResponse], error) {
	id, err := parseUUID("todo_list_id", req.Msg.TodoListId)
	if err != nil {
		return nil, err
	}

	q := query.ToEnvelope(appquery.GetTodoList{ID: todolist.ID(id)})

	tl, err := srv.GetTodoListHandler.Handle(ctx, q)
	if err != nil {
		return nil, mapCommandError("connect.GetTodoList", err)
	}

	return connect.NewResponse(&todolistv1.GetTodoListResponse{
		TodoList: protoconv.FromTodoList(tl),
	}), nil
}

// AddTodoItem implements the Connect service handler.
func (srv TodoListServiceServer) AddTodoItem(
	ctx context.Context,
	req *connect.Request[todolistv1.AddTodoItemRequest],
) (*connect.Response[emptypb.Empty], error) {
	listID, err := parseUUID("todo_list_id", req.Msg.TodoListId)
	if err != nil {
		return nil, err
	}

	itemID, err := parseUUID("todo_item_id", req.Msg.TodoItemId)
	if err != nil {
		return nil, err
	}

	var dueDate time.Time
	if req.Msg.DueDate != nil {
		dueDate = req.Msg.DueDate.AsTime()
	}

	cmd := command.ToEnvelope(appcommand.AddTodoListItem{
		TodoListID:  todolist.ID(listID),
		TodoItemID:  todolist.ItemID(itemID),
		Title:       req.Msg.Title,
		Description: req.Msg.Description,
		DueDate:     dueDate,
	})

	if err := srv.AddTodoListHandler.Handle(ctx, cmd); err != nil {
		return nil, mapCommandError("connect.AddTodoItem", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// MarkTodoItemAsDone implements the Connect service handler.
//
// Not wired up yet in the example: the corresponding command handler is not
// yet defined, so this returns Unimplemented to be explicit about it.
func (srv TodoListServiceServer) MarkTodoItemAsDone(
	_ context.Context,
	_ *connect.Request[todolistv1.MarkTodoItemAsDoneRequest],
) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("mark-as-done not implemented"))
}

// MarkTodoItemAsPending implements the Connect service handler.
//
// Not wired up yet in the example: see MarkTodoItemAsDone.
func (srv TodoListServiceServer) MarkTodoItemAsPending(
	_ context.Context,
	_ *connect.Request[todolistv1.MarkTodoItemAsPendingRequest],
) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("mark-as-pending not implemented"))
}

// DeleteTodoItem implements the Connect service handler.
//
// Not wired up yet in the example: see MarkTodoItemAsDone.
func (srv TodoListServiceServer) DeleteTodoItem(
	_ context.Context,
	_ *connect.Request[todolistv1.DeleteTodoItemRequest],
) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("delete not implemented"))
}

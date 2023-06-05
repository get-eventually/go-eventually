// Package grpc contains the gRPC server implementations for the application.
package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/query"
	todolistv1 "github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1"
	"github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1/todolistv1connect"
	appcommand "github.com/get-eventually/go-eventually/examples/todolist/internal/command"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/protoconv"
	appquery "github.com/get-eventually/go-eventually/examples/todolist/internal/query"
)

var _ todolistv1connect.TodoListServiceHandler = TodoListServiceServer{}

// TodoListServiceServer is the gRPC server implementation for this application.
type TodoListServiceServer struct {
	todolistv1connect.UnimplementedTodoListServiceHandler

	GenerateIDFunc func() uuid.UUID

	GetTodoListHandler appquery.GetTodoListHandler

	CreateTodoListHandler appcommand.CreateTodoListHandler
	AddTodoListHandler    appcommand.AddTodoListItemHandler
}

// GetTodoList implements todolistv1connect.TodoListServiceHandler.
func (srv TodoListServiceServer) GetTodoList(
	ctx context.Context,
	req *connect.Request[todolistv1.GetTodoListRequest],
) (*connect.Response[todolistv1.GetTodoListResponse], error) {
	id, err := uuid.Parse(req.Msg.TodoListId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("grpc.TodoListServiceServer: failed to parse todoListId param, %v", err))
	}

	q := query.ToEnvelope(appquery.GetTodoList{
		ID: todolist.ID(id),
	})

	makeError := func(code connect.Code, err error) *connect.Error {
		return connect.NewError(
			code,
			fmt.Errorf("grpc.TodoListServiceServer.GetTodoList: failed to handle query, %v", err),
		)
	}

	switch res, err := srv.GetTodoListHandler.Handle(ctx, q); {
	case err == nil:
		return connect.NewResponse(&todolistv1.GetTodoListResponse{
			TodoList: protoconv.FromTodoList(res),
		}), nil

	case errors.Is(err, aggregate.ErrRootNotFound):
		return nil, makeError(connect.CodeNotFound, err)

	default:
		return nil, makeError(connect.CodeInternal, err)
	}
}

// CreateTodoList implements todolistv1connect.TodoListServiceHandler.
func (srv TodoListServiceServer) CreateTodoList(
	ctx context.Context,
	req *connect.Request[todolistv1.CreateTodoListRequest],
) (*connect.Response[todolistv1.CreateTodoListResponse], error) {
	id := srv.GenerateIDFunc()

	cmd := command.ToEnvelope(appcommand.CreateTodoList{
		ID:    todolist.ID(id),
		Title: req.Msg.Title,
		Owner: req.Msg.Owner,
	})

	makeError := func(code connect.Code, err error) *connect.Error {
		return connect.NewError(
			code,
			fmt.Errorf("grpc.TodoListServiceServer.CreateTodoList: failed to handle command, %v", err),
		)
	}

	switch err := srv.CreateTodoListHandler.Handle(ctx, cmd); {
	case err == nil:
		return connect.NewResponse(&todolistv1.CreateTodoListResponse{
			TodoListId: id.String(),
		}), nil

	case errors.Is(err, todolist.ErrEmptyTitle), errors.Is(err, todolist.ErrNoOwnerSpecified):
		return nil, makeError(connect.CodeInvalidArgument, err)

	default:
		return nil, makeError(connect.CodeInternal, err)
	}
}

// AddTodoItem implements todolistv1connect.TodoListServiceHandler.
func (srv TodoListServiceServer) AddTodoItem(
	ctx context.Context,
	req *connect.Request[todolistv1.AddTodoItemRequest],
) (*connect.Response[todolistv1.AddTodoItemResponse], error) {
	todoListID, err := uuid.Parse(req.Msg.TodoListId)
	if err != nil {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("grpc.TodoListServiceServer.AddTodoItem: failed to parse todoListId into uuid, %v", err),
		)
	}

	id := srv.GenerateIDFunc()

	cmd := command.ToEnvelope(appcommand.AddTodoListItem{
		TodoListID:  todolist.ID(todoListID),
		TodoItemID:  todolist.ItemID(id),
		Title:       req.Msg.Title,
		Description: req.Msg.Description,
	})

	if req.Msg.DueDate != nil {
		cmd.Message.DueDate = req.Msg.DueDate.AsTime()
	}

	makeError := func(code connect.Code, err error) *connect.Error {
		return connect.NewError(
			code,
			fmt.Errorf("grpc.TodoListServiceServer.AddTodoItem: failed to handle command, %v", err),
		)
	}

	switch err := srv.AddTodoListHandler.Handle(ctx, cmd); {
	case err == nil:
		return connect.NewResponse(&todolistv1.AddTodoItemResponse{
			TodoItemId: id.String(),
		}), nil

	case errors.Is(err, todolist.ErrEmptyItemTitle):
		return nil, makeError(connect.CodeInvalidArgument, err)

	case errors.Is(err, todolist.ErrItemAlreadyExists):
		return nil, makeError(connect.CodeAlreadyExists, err)

	default:
		return nil, makeError(connect.CodeInternal, err)
	}
}

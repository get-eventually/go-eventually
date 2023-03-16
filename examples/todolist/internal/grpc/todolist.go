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

type TodoListServiceServer struct {
	todolistv1connect.UnimplementedTodoListServiceHandler

	GenerateIDFunc func() uuid.UUID

	GetTodoListHandler appquery.GetTodoListHandler

	CreateTodoListHandler appcommand.CreateTodoListHandler
	AddTodoListHandler    appcommand.AddTodoListItemHandler
}

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

	switch res, err := srv.GetTodoListHandler.Handle(ctx, q); {
	case err == nil:
		return connect.NewResponse(&todolistv1.GetTodoListResponse{
			TodoList: protoconv.FromTodoList(res),
		}), nil

	case errors.Is(err, aggregate.ErrRootNotFound):
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("grpc:TodoListServiceServer: failed to handle query, %v", err))

	default:
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("gprc.TodoListServiceServer: failed to handle query, %v", err))
	}
}

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

	switch err := srv.CreateTodoListHandler.Handle(ctx, cmd); {
	case err == nil:
		return connect.NewResponse(&todolistv1.CreateTodoListResponse{
			TodoListId: id.String(),
		}), nil

	case errors.Is(err, todolist.ErrEmptyTitle), errors.Is(err, todolist.ErrNoOwnerSpecified):
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("grpc.TodoListServiceServer.CreateTodoList: invalid arguments, %v", err),
		)

	default:
		return nil, connect.NewError(
			connect.CodeInternal,
			fmt.Errorf("grpc.TodoListServiceServer.CreateTodoList: failed to handle command, %v", err),
		)
	}
}

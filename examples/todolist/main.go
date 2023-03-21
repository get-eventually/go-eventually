// Package main contains the entrypoint for the TodoList gRPC API application.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	grpcreflect "github.com/bufbuild/connect-grpcreflect-go"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/test"
	"github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1/todolistv1connect"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/command"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/grpc"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/query"
)

type config struct {
	Server struct {
		Address      string        `default:":8080" required:"true"`
		ReadTimeout  time.Duration `default:"10s" required:"true"`
		WriteTimeout time.Duration `default:"10s" required:"true"`
	}
}

func parseConfig() (*config, error) {
	var config config

	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("config: failed to parse from env, %v", err)
	}

	return &config, nil
}

func run() error {
	config, err := parseConfig()
	if err != nil {
		return fmt.Errorf("todolist.main: failed to parse config, %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("todolist.main: failed to initialize logger, %v", err)
	}

	//nolint:errcheck // No need for this error to come up if it happens.
	defer logger.Sync()

	eventStore := test.NewInMemoryEventStore()
	todoListRepository := aggregate.NewEventSourcedRepository(eventStore, todolist.Type)

	todoListServiceServer := &grpc.TodoListServiceServer{
		GenerateIDFunc: uuid.New,
		GetTodoListHandler: query.GetTodoListHandler{
			Getter: todoListRepository,
		},
		CreateTodoListHandler: command.CreateTodoListHandler{
			Clock:      time.Now,
			Repository: todoListRepository,
		},
		AddTodoListHandler: command.AddTodoListItemHandler{
			Clock:      time.Now,
			Repository: todoListRepository,
		},
	}

	mux := http.NewServeMux()
	mux.Handle(todolistv1connect.NewTodoListServiceHandler(todoListServiceServer))
	mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker(todolistv1connect.TodoListServiceName)))
	mux.Handle(grpcreflect.NewHandlerV1(grpcreflect.NewStaticReflector(todolistv1connect.TodoListServiceName)))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(grpcreflect.NewStaticReflector(todolistv1connect.TodoListServiceName)))

	logger.Sugar().Infow("grpc server started",
		"address", config.Server.Address,
	)

	// TODO: implement graceful shutdown
	srv := &http.Server{
		Addr:         config.Server.Address,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
	}

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("todolist.main: grpc server exited with error, %v", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

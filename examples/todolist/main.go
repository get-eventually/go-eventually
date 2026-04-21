// Package main is the entrypoint for the TodoList Connect service example.
//
// The service is backed by an in-memory event.Store: state is lost on
// restart. This example is about showcasing how to wire the
// go-eventually building blocks together, not about persistence.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	connectgrpchealth "connectrpc.com/grpchealth"
	connectgrpcreflect "connectrpc.com/grpcreflect"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1/todolistv1connect"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/todolist"
)

type serverConfig struct {
	Address         string        `default:":8080" envconfig:"ADDRESS"`
	ReadTimeout     time.Duration `default:"10s"   envconfig:"READ_TIMEOUT"`
	WriteTimeout    time.Duration `default:"10s"   envconfig:"WRITE_TIMEOUT"`
	ShutdownTimeout time.Duration `default:"15s"   envconfig:"SHUTDOWN_TIMEOUT"`
}

type config struct {
	Server serverConfig
}

func parseConfig() (config, error) {
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		return config{}, fmt.Errorf("failed to parse config from env, %w", err)
	}

	return cfg, nil
}

func run() error { //nolint:funlen // Single linear wire-up of the service; splitting hurts readability.
	cfg, err := parseConfig()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   false,
		ReplaceAttr: nil,
	}))
	slog.SetDefault(logger)

	// In-memory plumbing: a single Store feeds both the command and query
	// sides through an EventSourcedRepository.
	eventStore := event.NewInMemoryStore()
	todoListRepository := aggregate.NewEventSourcedRepository(eventStore, todolist.Type)

	server := todolist.ConnectServiceHandler{
		UnimplementedTodoListServiceHandler: todolistv1connect.UnimplementedTodoListServiceHandler{},
		GetQueryHandler: todolist.GetQueryHandler{
			Getter: todoListRepository,
		},
		CreateCommandHandler: todolist.CreateCommandHandler{
			Clock:      time.Now,
			Repository: todoListRepository,
		},
		AddItemCommandHandler: todolist.AddItemCommandHandler{
			Clock:      time.Now,
			Repository: todoListRepository,
		},
	}

	mux := http.NewServeMux()
	mux.Handle(todolistv1connect.NewTodoListServiceHandler(server))
	mux.Handle(connectgrpchealth.NewHandler(
		connectgrpchealth.NewStaticChecker(todolistv1connect.TodoListServiceName),
	))
	mux.Handle(connectgrpcreflect.NewHandlerV1(
		connectgrpcreflect.NewStaticReflector(todolistv1connect.TodoListServiceName),
	))
	mux.Handle(connectgrpcreflect.NewHandlerV1Alpha(
		connectgrpcreflect.NewStaticReflector(todolistv1connect.TodoListServiceName),
	))

	srv := &http.Server{ //nolint:exhaustruct // Stdlib struct with many optional fields; defaults are fine.
		Addr:              cfg.Server.Address,
		Handler:           h2c.NewHandler(mux, &http2.Server{}), //nolint:exhaustruct // h2c.Server defaults are fine.
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: cfg.Server.ReadTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErrs := make(chan error, 1)

	go func() {
		logger.Info("connect server started", slog.String("address", cfg.Server.Address))

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrs <- fmt.Errorf("connect server exited unexpectedly, %w", err)

			return
		}

		serverErrs <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrs:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed, %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "todolist:", err)
		os.Exit(1)
	}
}

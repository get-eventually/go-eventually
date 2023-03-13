package command_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/test/scenario"
	"github.com/get-eventually/go-eventually/core/version"
	appcommand "github.com/get-eventually/go-eventually/examples/todolist/command"
	"github.com/get-eventually/go-eventually/examples/todolist/domain/todolist"
)

func TestCreateTodoListHandler(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	clock := func() time.Time { return now }

	commandHandlerFactory := func(s event.Store) appcommand.CreateTodoListHandler {
		return appcommand.CreateTodoListHandler{
			Clock:      clock,
			Repository: aggregate.NewEventSourcedRepository(s, todolist.Type),
		}
	}

	t.Run("it fails when an invalid id has been provided", func(t *testing.T) {
		scenario.CommandHandler[appcommand.CreateTodoList, appcommand.CreateTodoListHandler]().
			When(command.ToEnvelope(appcommand.CreateTodoList{
				ID:    todolist.ID(uuid.Nil),
				Title: "my-title",
				Owner: "owner",
			})).
			ThenError(todolist.ErrEmptyID).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when a title has not been provided", func(t *testing.T) {
		scenario.CommandHandler[appcommand.CreateTodoList, appcommand.CreateTodoListHandler]().
			When(command.ToEnvelope(appcommand.CreateTodoList{
				ID:    todolist.ID(id),
				Title: "",
				Owner: "owner",
			})).
			ThenError(todolist.ErrEmptyTitle).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when an owner has not been provided", func(t *testing.T) {
		scenario.CommandHandler[appcommand.CreateTodoList, appcommand.CreateTodoListHandler]().
			When(command.ToEnvelope(appcommand.CreateTodoList{
				ID:    todolist.ID(id),
				Title: "my-title",
				Owner: "",
			})).
			ThenError(todolist.ErrNoOwnerSpecified).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it works", func(t *testing.T) {
		scenario.CommandHandler[appcommand.CreateTodoList, appcommand.CreateTodoListHandler]().
			When(command.ToEnvelope(appcommand.CreateTodoList{
				ID:    todolist.ID(id),
				Title: "my-title",
				Owner: "owner",
			})).
			Then(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todolist.ID(id),
					Title:        "my-title",
					Owner:        "owner",
					CreationTime: now,
				}),
			}).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when trying to create a TodoList that exists already", func(t *testing.T) {
		scenario.CommandHandler[appcommand.CreateTodoList, appcommand.CreateTodoListHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todolist.ID(id),
					Title:        "my-title",
					Owner:        "owner",
					CreationTime: now,
				}),
			}).
			When(command.ToEnvelope(appcommand.CreateTodoList{
				ID:    todolist.ID(id),
				Title: "my-title",
				Owner: "owner",
			})).
			ThenError(version.ConflictError{
				Expected: 0,
				Actual:   1,
			}).
			AssertOn(t, commandHandlerFactory)
	})
}

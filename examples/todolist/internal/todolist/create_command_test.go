package todolist_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/command"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/todolist"
)

func TestCreateCommandHandler(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	clock := func() time.Time { return now }

	commandHandlerFactory := func(s event.Store) todolist.CreateCommandHandler {
		return todolist.CreateCommandHandler{
			Clock:      clock,
			Repository: aggregate.NewEventSourcedRepository(s, todolist.Type),
		}
	}

	t.Run("it fails when an invalid id has been provided", func(t *testing.T) {
		command.Scenario[todolist.CreateCommand, todolist.CreateCommandHandler]().
			When(command.ToEnvelope(todolist.CreateCommand{
				ID:    todolist.ID(uuid.Nil),
				Title: "my-title",
				Owner: "owner",
			})).
			ThenError(todolist.ErrEmptyID).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when a title has not been provided", func(t *testing.T) {
		command.Scenario[todolist.CreateCommand, todolist.CreateCommandHandler]().
			When(command.ToEnvelope(todolist.CreateCommand{
				ID:    todolist.ID(id),
				Title: "",
				Owner: "owner",
			})).
			ThenError(todolist.ErrEmptyTitle).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when an owner has not been provided", func(t *testing.T) {
		command.Scenario[todolist.CreateCommand, todolist.CreateCommandHandler]().
			When(command.ToEnvelope(todolist.CreateCommand{
				ID:    todolist.ID(id),
				Title: "my-title",
				Owner: "",
			})).
			ThenError(todolist.ErrNoOwnerSpecified).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it works", func(t *testing.T) {
		command.Scenario[todolist.CreateCommand, todolist.CreateCommandHandler]().
			When(command.ToEnvelope(todolist.CreateCommand{
				ID:    todolist.ID(id),
				Title: "my-title",
				Owner: "owner",
			})).
			Then(event.Persisted{
				StreamID: event.StreamID(todolist.ID(id).String()),
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
}

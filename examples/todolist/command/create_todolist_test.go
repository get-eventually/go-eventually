package command_test

import (
	"testing"
	"time"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/test/scenario"
	"github.com/google/uuid"

	appcommand "github.com/get-eventually/go-eventually/examples/todolist/command"
	"github.com/get-eventually/go-eventually/examples/todolist/domain/todolist"
)

func TestCreateTodoListHandler(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	clock := func() time.Time { return now }

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
			AssertOn(t, func(s event.Store) appcommand.CreateTodoListHandler {
				return appcommand.CreateTodoListHandler{
					Clock:      clock,
					Repository: aggregate.NewEventSourcedRepository(s, todolist.Type),
				}
			})
	})
}

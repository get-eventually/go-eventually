package command_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/test/scenario"
	appcommand "github.com/get-eventually/go-eventually/examples/todolist/internal/command"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
)

func TestAddTodoListItem(t *testing.T) {
	now := time.Now()
	commandHandlerFactory := func(es event.Store) appcommand.AddTodoListItemHandler {
		return appcommand.AddTodoListItemHandler{
			Clock:      func() time.Time { return now },
			Repository: aggregate.NewEventSourcedRepository(es, todolist.Type),
		}
	}

	todoListID := todolist.ID(uuid.New())
	todoItemID := todolist.ItemID(uuid.New())
	listTitle := "my list"
	listOwner := "me"

	t.Run("it fails when the target TodoList does not exist", func(t *testing.T) {
		scenario.CommandHandler[appcommand.AddTodoListItem, appcommand.AddTodoListItemHandler]().
			When(command.ToEnvelope(appcommand.AddTodoListItem{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
				Title:      "a todo item that should fail",
			})).
			ThenError(aggregate.ErrRootNotFound).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when the same item has already been added", func(t *testing.T) {
		scenario.CommandHandler[appcommand.AddTodoListItem, appcommand.AddTodoListItemHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        listTitle,
					Owner:        listOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}, event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  2,
				Envelope: event.ToEnvelope(todolist.ItemWasAdded{
					ID:           todoItemID,
					Title:        "a todo item that should succeed",
					CreationTime: now,
				}),
			}).
			When(command.ToEnvelope(appcommand.AddTodoListItem{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
				Title:      "uh oh, this is gonna fail",
			})).
			ThenError(todolist.ErrItemAlreadyExists).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when the item id provided is empty", func(t *testing.T) {
		scenario.CommandHandler[appcommand.AddTodoListItem, appcommand.AddTodoListItemHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        listTitle,
					Owner:        listOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}).
			When(command.ToEnvelope(appcommand.AddTodoListItem{
				TodoListID: todoListID,
				TodoItemID: todolist.ItemID(uuid.Nil),
				Title:      "i think i forgot to add an id...",
			})).
			ThenError(todolist.ErrEmptyItemID).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when an empty item title is provided", func(t *testing.T) {
		scenario.CommandHandler[appcommand.AddTodoListItem, appcommand.AddTodoListItemHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        listTitle,
					Owner:        listOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}).
			When(command.ToEnvelope(appcommand.AddTodoListItem{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
				Title:      "",
			})).
			ThenError(todolist.ErrEmptyItemTitle).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it works", func(t *testing.T) {
		scenario.CommandHandler[appcommand.AddTodoListItem, appcommand.AddTodoListItemHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        listTitle,
					Owner:        listOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}).
			When(command.ToEnvelope(appcommand.AddTodoListItem{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
				Title:      "a todo item that should succeed",
			})).
			Then(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  2,
				Envelope: event.ToEnvelope(todolist.ItemWasAdded{
					ID:           todoItemID,
					Title:        "a todo item that should succeed",
					CreationTime: now,
				}),
			}).
			AssertOn(t, commandHandlerFactory)
	})
}

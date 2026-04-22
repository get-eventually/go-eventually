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

func TestAddItemCommandHandler(t *testing.T) {
	now := time.Now()
	commandHandlerFactory := func(es event.Store) todolist.AddItemCommandHandler {
		return todolist.AddItemCommandHandler{
			Clock:      func() time.Time { return now },
			Repository: aggregate.NewEventSourcedRepository(es, todolist.Type),
		}
	}

	todoListID := todolist.ID(uuid.New())
	todoItemID := todolist.ItemID(uuid.New())

	t.Run("it fails when the target TodoList does not exist", func(t *testing.T) {
		command.Scenario[todolist.AddItemCommand, todolist.AddItemCommandHandler]().
			When(command.ToEnvelope(todolist.AddItemCommand{
				TodoListID:  todoListID,
				TodoItemID:  todoItemID,
				Title:       "a todo item that should fail",
				Description: "",
				DueDate:     time.Time{},
			})).
			ThenError(aggregate.ErrRootNotFound).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when the same item has already been added", func(t *testing.T) {
		command.Scenario[todolist.AddItemCommand, todolist.AddItemCommandHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        testListTitle,
					Owner:        testListOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}, event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  2,
				Envelope: event.ToEnvelope(todolist.ItemWasAdded{
					ID:           todoItemID,
					Title:        "a todo item that should succeed",
					Description:  "",
					DueDate:      time.Time{},
					CreationTime: now,
				}),
			}).
			When(command.ToEnvelope(todolist.AddItemCommand{
				TodoListID:  todoListID,
				TodoItemID:  todoItemID,
				Title:       "uh oh, this is gonna fail",
				Description: "",
				DueDate:     time.Time{},
			})).
			ThenError(todolist.ErrItemAlreadyExists).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when the item title is empty", func(t *testing.T) {
		command.Scenario[todolist.AddItemCommand, todolist.AddItemCommandHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        testListTitle,
					Owner:        testListOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}).
			When(command.ToEnvelope(todolist.AddItemCommand{
				TodoListID:  todoListID,
				TodoItemID:  todoItemID,
				Title:       "",
				Description: "",
				DueDate:     time.Time{},
			})).
			ThenError(todolist.ErrEmptyItemTitle).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it works", func(t *testing.T) {
		command.Scenario[todolist.AddItemCommand, todolist.AddItemCommandHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  1,
				Envelope: event.ToEnvelope(todolist.WasCreated{
					ID:           todoListID,
					Title:        testListTitle,
					Owner:        testListOwner,
					CreationTime: now.Add(-2 * time.Minute),
				}),
			}).
			When(command.ToEnvelope(todolist.AddItemCommand{
				TodoListID:  todoListID,
				TodoItemID:  todoItemID,
				Title:       "a todo item that should succeed",
				Description: "",
				DueDate:     time.Time{},
			})).
			Then(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  2,
				Envelope: event.ToEnvelope(todolist.ItemWasAdded{
					ID:           todoItemID,
					Title:        "a todo item that should succeed",
					Description:  "",
					DueDate:      time.Time{},
					CreationTime: now,
				}),
			}).
			AssertOn(t, commandHandlerFactory)
	})
}

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

func TestDeleteItemCommandHandler(t *testing.T) {
	now := time.Now()
	commandHandlerFactory := func(es event.Store) todolist.DeleteItemCommandHandler {
		return todolist.DeleteItemCommandHandler{
			Repository: aggregate.NewEventSourcedRepository(es, todolist.Type),
		}
	}

	todoListID := todolist.ID(uuid.New())
	todoItemID := todolist.ItemID(uuid.New())

	listCreated := event.Persisted{
		StreamID: event.StreamID(todoListID.String()),
		Version:  1,
		Envelope: event.ToEnvelope(todolist.WasCreated{
			ID:           todoListID,
			Title:        testListTitle,
			Owner:        testListOwner,
			CreationTime: now.Add(-2 * time.Minute),
		}),
	}
	itemAdded := event.Persisted{
		StreamID: event.StreamID(todoListID.String()),
		Version:  2,
		Envelope: event.ToEnvelope(todolist.ItemWasAdded{
			ID:           todoItemID,
			Title:        "buy groceries",
			Description:  "",
			DueDate:      time.Time{},
			CreationTime: now.Add(-time.Minute),
		}),
	}

	t.Run("it fails when the target TodoList does not exist", func(t *testing.T) {
		command.Scenario[todolist.DeleteItemCommand, todolist.DeleteItemCommandHandler]().
			When(command.ToEnvelope(todolist.DeleteItemCommand{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
			})).
			ThenError(aggregate.ErrRootNotFound).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when the item is not in the list", func(t *testing.T) {
		command.Scenario[todolist.DeleteItemCommand, todolist.DeleteItemCommandHandler]().
			Given(listCreated).
			When(command.ToEnvelope(todolist.DeleteItemCommand{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
			})).
			ThenError(todolist.ErrItemNotFound).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it fails when the item ID is empty", func(t *testing.T) {
		command.Scenario[todolist.DeleteItemCommand, todolist.DeleteItemCommandHandler]().
			Given(listCreated).
			When(command.ToEnvelope(todolist.DeleteItemCommand{
				TodoListID: todoListID,
				TodoItemID: todolist.ItemID(uuid.Nil),
			})).
			ThenError(todolist.ErrEmptyItemID).
			AssertOn(t, commandHandlerFactory)
	})

	t.Run("it works", func(t *testing.T) {
		command.Scenario[todolist.DeleteItemCommand, todolist.DeleteItemCommandHandler]().
			Given(listCreated, itemAdded).
			When(command.ToEnvelope(todolist.DeleteItemCommand{
				TodoListID: todoListID,
				TodoItemID: todoItemID,
			})).
			Then(event.Persisted{
				StreamID: event.StreamID(todoListID.String()),
				Version:  3,
				Envelope: event.ToEnvelope(todolist.ItemWasDeleted{
					ID: todoItemID,
				}),
			}).
			AssertOn(t, commandHandlerFactory)
	})
}

package todolist_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/test/scenario"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
)

func TestTodoList(t *testing.T) {
	t.Run("it works", func(t *testing.T) {
		now := time.Now()
		todoListID := todolist.ID(uuid.New())
		todoItemID := todolist.ItemID(uuid.New())

		scenario.AggregateRoot(todolist.Type).
			When(func() (*todolist.TodoList, error) {
				tl, err := todolist.Create(todoListID, "test list", "me", now)
				if err != nil {
					return nil, err
				}

				if err := tl.AddItem(todoItemID, "do something", "", time.Time{}, now); err != nil {
					return nil, err
				}

				if err := tl.MarkItemAsDone(todoItemID); err != nil {
					return nil, err
				}

				if err := tl.DeleteItem(todoItemID); err != nil {
					return nil, err
				}

				return tl, nil
			}).
			Then(4, event.ToEnvelope(todolist.WasCreated{
				ID:           todoListID,
				Title:        "test list",
				Owner:        "me",
				CreationTime: now,
			}), event.ToEnvelope(todolist.ItemWasAdded{
				ID:           todoItemID,
				Title:        "do something",
				CreationTime: now,
			}), event.ToEnvelope(todolist.ItemMarkedAsDone{
				ID: todoItemID,
			}), event.ToEnvelope(todolist.ItemWasDeleted{
				ID: todoItemID,
			})).
			AssertOn(t)
	})
}

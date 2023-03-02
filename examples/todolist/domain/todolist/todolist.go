package todolist

import (
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/google/uuid"
)

type ID uuid.UUID

func (id ID) String() string { return uuid.UUID(id).String() }

var Type = aggregate.Type[ID, *TodoList]{
	Name:    "TodoList",
	Factory: func() *TodoList { return new(TodoList) },
}

type TodoList struct {
	aggregate.BaseRoot

	id           ID
	title        string
	owner        string
	creationTime time.Time
	items        []*Item
}

// AggregateID implements aggregate.Root
func (tl *TodoList) AggregateID() ID {
	return tl.id
}

// Apply implements aggregate.Root
func (tl *TodoList) Apply(event event.Event) error {
	switch evt := event.(type) {
	case WasCreated:
		tl.id = evt.ID
		tl.title = evt.Title
		tl.owner = evt.Owner
		tl.creationTime = evt.CreationTime

	case ItemWasAdded:
		item := &Item{}
		if err := item.Apply(evt); err != nil {
			return fmt.Errorf("todolist.TodoList.Apply: failed to apply item event, %w", err)
		}
		tl.items = append(tl.items, item)

	case ItemMarkedAsDone:
	case ItemWasDeleted:

	default:
		return fmt.Errorf("todolist.TodoList.Apply: invalid event, %T", evt)
	}

	return nil
}

func Create(id ID, title, owner string, now time.Time) (*TodoList, error) {
	if uuid.UUID(id) == uuid.Nil {
		return nil, fmt.Errorf("invalid id")
	}

	if title == "" {
		return nil, fmt.Errorf("empty title")
	}

	if owner == "" {
		return nil, fmt.Errorf("empty owner")
	}

	var todoList TodoList
	if err := aggregate.RecordThat[ID](&todoList, event.ToEnvelope(WasCreated{
		ID:           id,
		Title:        title,
		Owner:        owner,
		CreationTime: now,
	})); err != nil {
		return nil, fmt.Errorf("todolist.Create: failed to apply domain event, %w", err)
	}

	return &todoList, nil
}

package todolist

import (
	"errors"
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

func (tl *TodoList) findItemByID(id ItemID) *Item {
	for _, item := range tl.items {
		if item.id == id {
			return item
		}
	}

	return nil
}

func (tl *TodoList) applyItemEvent(id ItemID, evt event.Event) error {
	item := tl.findItemByID(id)
	if item == nil {
		return fmt.Errorf("todolist.TodoList.Apply: item not found")
	}

	if err := item.Apply(evt); err != nil {
		return fmt.Errorf("todolist.TodoList.Apply: failed to apply item event, %w", err)
	}

	return nil
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

	case ItemMarkedAsPending:
		return tl.applyItemEvent(evt.ID, evt)

	case ItemMarkedAsDone:
		return tl.applyItemEvent(evt.ID, evt)

	case ItemWasDeleted:
		var items []*Item
		for _, item := range tl.items {
			if item.id == evt.ID {
				continue
			}

			items = append(items, item)
		}

		tl.items = items

	default:
		return fmt.Errorf("todolist.TodoList.Apply: invalid event, %T", evt)
	}

	return nil
}

var (
	ErrEmptyID           = errors.New("todolist.TodoList: empty id provided")
	ErrEmptyTitle        = errors.New("todolist.TodoList: empty title provided")
	ErrNoOwnerSpecified  = errors.New("todolist.TodoList: no owner specified")
	ErrEmptyItemID       = errors.New("todolist.TodoList: empty item id provided")
	ErrEmptyItemTitle    = errors.New("todolist.TodoList: empty item title provided")
	ErrItemAlreadyExists = errors.New("todolist.TodoList: item already exists")
)

func Create(id ID, title, owner string, now time.Time) (*TodoList, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("todolist.Create: failed to create new TodoList, %w", err)
	}

	if uuid.UUID(id) == uuid.Nil {
		return nil, wrapErr(ErrEmptyID)
	}

	if title == "" {
		return nil, wrapErr(ErrEmptyTitle)
	}

	if owner == "" {
		return nil, wrapErr(ErrNoOwnerSpecified)
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

func (todoList *TodoList) itemByID(id ItemID) (*Item, bool) {
	for _, item := range todoList.items {
		if item.id == id {
			return item, true
		}
	}

	return nil, false
}

func (todoList *TodoList) AddItem(id ItemID, title, description string, dueDate, now time.Time) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("todolist.AddItem: failed to add new TodoItem to list, %w", err)
	}

	if uuid.UUID(id) == uuid.Nil {
		return wrapErr(ErrEmptyItemID)
	}

	if title == "" {
		return wrapErr(ErrEmptyItemTitle)
	}

	if _, ok := todoList.itemByID(id); ok {
		return wrapErr(ErrItemAlreadyExists)
	}

	if err := aggregate.RecordThat[ID](todoList, event.ToEnvelope(ItemWasAdded{
		ID:           id,
		Title:        title,
		Description:  description,
		DueDate:      dueDate,
		CreationTime: now,
	})); err != nil {
		return fmt.Errorf("todolist.AddItem: failed to apply domain event, %w", err)
	}

	return nil
}

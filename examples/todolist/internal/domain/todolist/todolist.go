// Package todolist contains the domain types and implementations
// for the TodoList Aggregate Root.
package todolist

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
)

// ID is the unique identifier for a TodoList.
type ID uuid.UUID

func (id ID) String() string { return uuid.UUID(id).String() }

// Type represents the Aggregate Root type for usage with go-eventually utilities.
var Type = aggregate.Type[ID, *TodoList]{
	Name:    "TodoList",
	Factory: func() *TodoList { return new(TodoList) },
}

// TodoList is a list of different Todo items, that belongs to a specific owner.
type TodoList struct {
	aggregate.BaseRoot

	ID           ID
	Title        string
	Owner        string
	CreationTime time.Time
	Items        []*Item
}

// AggregateID implements aggregate.Root.
func (tl *TodoList) AggregateID() ID {
	return tl.ID
}

func (tl *TodoList) itemByID(id ItemID) (*Item, bool) {
	for _, item := range tl.Items {
		if item.ID == id {
			return item, true
		}
	}

	return nil, false
}

func (tl *TodoList) applyItemEvent(id ItemID, evt event.Event) error {
	item, ok := tl.itemByID(id)
	if !ok {
		return fmt.Errorf("todolist.TodoList.Apply: item not found")
	}

	if err := item.Apply(evt); err != nil {
		return fmt.Errorf("todolist.TodoList.Apply: failed to apply item event, %w", err)
	}

	return nil
}

// Apply implements aggregate.Root.
func (tl *TodoList) Apply(event event.Event) error {
	switch evt := event.(type) {
	case WasCreated:
		tl.ID = evt.ID
		tl.Title = evt.Title
		tl.Owner = evt.Owner
		tl.CreationTime = evt.CreationTime

	case ItemWasAdded:
		item := &Item{}
		if err := item.Apply(evt); err != nil {
			return fmt.Errorf("todolist.TodoList.Apply: failed to apply item event, %w", err)
		}

		tl.Items = append(tl.Items, item)

	case ItemMarkedAsPending:
		return tl.applyItemEvent(evt.ID, evt)

	case ItemMarkedAsDone:
		return tl.applyItemEvent(evt.ID, evt)

	case ItemWasDeleted:
		var items []*Item

		for _, item := range tl.Items {
			if item.ID == evt.ID {
				continue
			}

			items = append(items, item)
		}

		tl.Items = items

	default:
		return fmt.Errorf("todolist.TodoList.Apply: invalid event, %T", evt)
	}

	return nil
}

// Errors that can be returned by domain commands on a TodoList instance.
var (
	ErrEmptyID           = errors.New("todolist.TodoList: empty id provided")
	ErrEmptyTitle        = errors.New("todolist.TodoList: empty title provided")
	ErrNoOwnerSpecified  = errors.New("todolist.TodoList: no owner specified")
	ErrEmptyItemID       = errors.New("todolist.TodoList: empty item id provided")
	ErrEmptyItemTitle    = errors.New("todolist.TodoList: empty item title provided")
	ErrItemAlreadyExists = errors.New("todolist.TodoList: item already exists")
	ErrItemNotFound      = errors.New("todolist.TodoList: item was not found in list")
)

// Create creates a new TodoList.
//
// Both id, title and owner are required parameters: when empty, the function
// will return an error.
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

// AddItem adds a new Todo item to an existing list.
//
// Both id and title cannot be empty: if so, the method will return an error.
//
// Moreover, if the specified id is already being used by another Todo item,
// the method will return ErrItemAlreadyExists.
func (tl *TodoList) AddItem(id ItemID, title, description string, dueDate, now time.Time) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("todolist.AddItem: failed to add new TodoItem to list, %w", err)
	}

	if uuid.UUID(id) == uuid.Nil {
		return wrapErr(ErrEmptyItemID)
	}

	if title == "" {
		return wrapErr(ErrEmptyItemTitle)
	}

	if _, ok := tl.itemByID(id); ok {
		return wrapErr(ErrItemAlreadyExists)
	}

	if err := aggregate.RecordThat[ID](tl, event.ToEnvelope(ItemWasAdded{
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

func (tl *TodoList) recordItemEvent(id ItemID, eventFactory func() event.Envelope) error {
	if uuid.UUID(id) == uuid.Nil {
		return ErrEmptyItemID
	}

	if _, ok := tl.itemByID(id); !ok {
		return ErrItemNotFound
	}

	return aggregate.RecordThat[ID](tl, eventFactory())
}

// MarkItemAsDone marks the Todo item with the specified id as "done".
//
// The method returns an error when the id is empty, or it doesn't point
// to an existing Todo item.
func (tl *TodoList) MarkItemAsDone(id ItemID) error {
	err := tl.recordItemEvent(id, func() event.Envelope {
		return event.ToEnvelope(ItemMarkedAsDone{ID: id})
	})
	if err != nil {
		return fmt.Errorf("todolist.MarkItemAsDone: failed to mark item as done, %w", err)
	}

	return nil
}

// MarkItemAsPending marks the Todo item with the specified id as "pending".
//
// The method returns an error when the id is empty, or it doesn't point
// to an existing Todo item.
func (tl *TodoList) MarkItemAsPending(id ItemID) error {
	err := tl.recordItemEvent(id, func() event.Envelope {
		return event.ToEnvelope(ItemMarkedAsPending{ID: id})
	})
	if err != nil {
		return fmt.Errorf("todolist.MarkItemAsPending: failed to mark item as pending, %w", err)
	}

	return nil
}

// DeleteItem deletes the Todo item with the specified id from the TodoList.
//
// The method returns an error when the id is empty, or it doesn't point
// to an existing Todo item.
func (tl *TodoList) DeleteItem(id ItemID) error {
	err := tl.recordItemEvent(id, func() event.Envelope {
		return event.ToEnvelope(ItemWasDeleted{ID: id})
	})
	if err != nil {
		return fmt.Errorf("todolist.DeleteItem: failed to delete item, %w", err)
	}

	return nil
}

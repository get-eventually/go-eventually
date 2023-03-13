package todolist

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
)

// ItemID is the unique identifier type for a Todo Item.
type ItemID uuid.UUID

func (id ItemID) String() string { return uuid.UUID(id).String() }

// Item represents a Todo Item.
// Items are managed by a TodoList aggregate root instance.
type Item struct {
	aggregate.BaseRoot

	id           ItemID
	title        string
	description  string
	completed    bool
	dueDate      time.Time
	creationTime time.Time
}

// Apply implements aggregate.Root.
func (item *Item) Apply(event event.Event) error {
	switch evt := event.(type) {
	case ItemWasAdded:
		item.id = evt.ID
		item.title = evt.Title
		item.description = evt.Description
		item.completed = false
		item.dueDate = evt.DueDate
		item.creationTime = evt.CreationTime

	case ItemMarkedAsDone:
		item.completed = true

	case ItemMarkedAsPending:
		item.completed = false

	default:
		return fmt.Errorf("todolist.Item.Apply: unsupported event, %T", evt)
	}

	return nil
}

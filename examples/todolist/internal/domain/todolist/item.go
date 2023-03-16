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

	ID           ItemID
	Title        string
	Description  string
	Completed    bool
	DueDate      time.Time
	CreationTime time.Time
}

// Apply implements aggregate.Root.
func (item *Item) Apply(event event.Event) error {
	switch evt := event.(type) {
	case ItemWasAdded:
		item.ID = evt.ID
		item.Title = evt.Title
		item.Description = evt.Description
		item.Completed = false
		item.DueDate = evt.DueDate
		item.CreationTime = evt.CreationTime

	case ItemMarkedAsDone:
		item.Completed = true

	case ItemMarkedAsPending:
		item.Completed = false

	default:
		return fmt.Errorf("todolist.Item.Apply: unsupported event, %T", evt)
	}

	return nil
}

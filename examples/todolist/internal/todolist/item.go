package todolist

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
)

// ItemID is the unique identifier type for a Todo Item.
type ItemID uuid.UUID

// String returns the canonical UUID string representation of the ItemID.
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

// Apply implements aggregate.Aggregate.
func (item *Item) Apply(evt event.Event) error {
	switch evt := evt.(type) {
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

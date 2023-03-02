package todolist

import (
	"fmt"
	"time"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/google/uuid"
)

type ItemID uuid.UUID

func (id ItemID) String() string { return uuid.UUID(id).String() }

type Item struct {
	aggregate.BaseRoot

	id          ItemID
	title       string
	description string
	completed   bool
	dueDate     time.Time
}

func (item *Item) Apply(event event.Event) error {
	switch evt := event.(type) {
	case ItemWasAdded:
		item.id = evt.ID
		item.title = evt.Title
		item.description = evt.Description
		item.completed = false
		item.dueDate = evt.DueDate
	default:
		return fmt.Errorf("todolist.Item.Apply: unsupported event, %T", evt)
	}

	return nil
}

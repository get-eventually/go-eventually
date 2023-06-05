package todolist

import "time"

// WasCreated is the Domain Event issued when new TodoList gets created.
type WasCreated struct {
	ID           ID
	Title        string
	Owner        string
	CreationTime time.Time
}

// Name implements message.Message.
func (WasCreated) Name() string { return "TodoListWasCreated" }

// ItemWasAdded is the Domain Event issued when a new Item gets added
// to an existing TodoList.
type ItemWasAdded struct {
	ID           ItemID
	Title        string
	Description  string
	DueDate      time.Time
	CreationTime time.Time
}

// Name implements message.Message.
func (ItemWasAdded) Name() string { return "TodoListItemWasAdded" }

// ItemMarkedAsDone is the Domain Event issued when an existing Item
// in a TodoList gets marked as "done", or "completed".
type ItemMarkedAsDone struct {
	ID ItemID
}

// Name implements message.Message.
func (ItemMarkedAsDone) Name() string { return "TodoListItemMarkedAsDone" }

// ItemMarkedAsPending is the Domain Event issued when an existing Item
// in a TodoList gets marked as "pending".
type ItemMarkedAsPending struct {
	ID ItemID
}

// Name implements message.Message.
func (ItemMarkedAsPending) Name() string { return "TodoListItemMarkedAsPending" }

// ItemWasDeleted is the Domain Event issued when an existing Item
// gets deleted from a TodoList.
type ItemWasDeleted struct {
	ID ItemID
}

// Name implements message.Message.
func (ItemWasDeleted) Name() string { return "TodoListItemWasDeleted" }

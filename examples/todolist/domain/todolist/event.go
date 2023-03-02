package todolist

import "time"

type WasCreated struct {
	ID           ID
	Title        string
	Owner        string
	CreationTime time.Time
}

func (WasCreated) Name() string { return "TodoListWasCreated" }

type ItemWasAdded struct {
	ID          ItemID
	Title       string
	Description string
	DueDate     time.Time
}

func (ItemWasAdded) Name() string { return "TodoListItemWasAdded" }

type ItemMarkedAsDone struct{}

func (ItemMarkedAsDone) Name() string { return "TodoListItemMarkedAsDone" }

type ItemWasDeleted struct{}

func (ItemWasDeleted) Name() string { return "TodoListItemWasDeleted" }

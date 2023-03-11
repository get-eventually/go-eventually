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
	ID           ItemID
	Title        string
	Description  string
	DueDate      time.Time
	CreationTime time.Time
}

func (ItemWasAdded) Name() string { return "TodoListItemWasAdded" }

type ItemMarkedAsDone struct {
	ID ItemID
}

func (ItemMarkedAsDone) Name() string { return "TodoListItemMarkedAsDone" }

type ItemMarkedAsPending struct {
	ID ItemID
}

func (ItemMarkedAsPending) Name() string { return "TodoListItemMarkedAsPending" }

type ItemWasDeleted struct {
	ID ItemID
}

func (ItemWasDeleted) Name() string { return "TodoListItemWasDeleted" }

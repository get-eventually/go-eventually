package todolist

import "github.com/get-eventually/go-eventually/core/aggregate"

type (
	// Getter is a helper type for an aggregate.Getter interface for a TodoList.
	Getter = aggregate.Getter[ID, *TodoList]

	// Saver is a helper type for an aggregate.Saver interface for a TodoList.
	Saver = aggregate.Saver[ID, *TodoList]

	// Repository is a helper type for an aggregate.Repository interface for a TodoList.
	Repository = aggregate.Repository[ID, *TodoList]
)

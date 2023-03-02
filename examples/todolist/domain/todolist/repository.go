package todolist

import "github.com/get-eventually/go-eventually/core/aggregate"

type (
	Getter     = aggregate.Getter[ID, *TodoList]
	Saver      = aggregate.Saver[ID, *TodoList]
	Repository = aggregate.Repository[ID, *TodoList]
)

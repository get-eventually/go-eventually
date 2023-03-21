// Package protoconv contains methods for conversion from Protobufs to Domain Objects and back.
package protoconv

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	todolistv1 "github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/domain/todolist"
)

// FromTodoList converts a TodoList aggregate root into its Protobuf counterpart.
func FromTodoList(tl *todolist.TodoList) *todolistv1.TodoList {
	result := &todolistv1.TodoList{
		Id:           tl.ID.String(),
		Title:        tl.Title,
		Owner:        tl.Owner,
		CreationTime: timestamppb.New(tl.CreationTime),
	}

	for _, item := range tl.Items {
		ritem := &todolistv1.TodoItem{
			Id:           item.ID.String(),
			Title:        item.Title,
			Description:  item.Description,
			Completed:    item.Completed,
			CreationTime: timestamppb.New(item.CreationTime),
		}

		if !item.DueDate.IsZero() {
			ritem.DueDate = timestamppb.New(item.DueDate)
		}

		result.Items = append(result.Items, ritem)
	}

	return result
}

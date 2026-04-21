// Package protoconv contains conversions between domain types and their
// generated Protobuf counterparts.
package protoconv

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	todolistv1 "github.com/get-eventually/go-eventually/examples/todolist/gen/todolist/v1"
	"github.com/get-eventually/go-eventually/examples/todolist/internal/todolist"
)

// FromTodoList converts a TodoList aggregate root into its Protobuf counterpart.
func FromTodoList(tl *todolist.TodoList) *todolistv1.TodoList {
	result := &todolistv1.TodoList{
		Id:           tl.ID.String(),
		Title:        tl.Title,
		Owner:        tl.Owner,
		CreationTime: timestamppb.New(tl.CreationTime),
		Items:        make([]*todolistv1.TodoItem, 0, len(tl.Items)),
	}

	for _, item := range tl.Items {
		pbItem := &todolistv1.TodoItem{
			Id:           item.ID.String(),
			Title:        item.Title,
			Description:  item.Description,
			Completed:    item.Completed,
			DueDate:      nil,
			CreationTime: timestamppb.New(item.CreationTime),
		}

		if !item.DueDate.IsZero() {
			pbItem.DueDate = timestamppb.New(item.DueDate)
		}

		result.Items = append(result.Items, pbItem)
	}

	return result
}

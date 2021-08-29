package command

import (
	"context"

	"github.com/get-eventually/go-eventually"
)

// Command is a marker interface that represents a Domain Command.
type Command eventually.Payload

// Handler is the interface that defines a Command Handler, component
// that receives a specific kind of Command and executes the business logic
// related to that particular Command.
type Handler interface {
	// CommandType returns an empty instance of the Command type the Handler
	// is supposed to handle.
	CommandType() Command

	// Handle receives a Domain Command and executes the necessary business logic.
	//
	// Since Commands' responsibility is only to trigger side effects, and not
	// answering Domain Queries, the Handle method only returns an error, whether
	// the Command has failed execution or not.
	Handle(context.Context, eventually.Command) error
}

var _ Handler = HandlerFunc(nil)

// HandlerFunc is a functional Command Handler.
type HandlerFunc func(context.Context, eventually.Command) error

// CommandType returns nil.
//
// Make sure not to use HandlerFunc in a command.Dispatcher, because it will
// probably require this function to return a meaningful value.
func (fn HandlerFunc) CommandType() Command {
	return nil
}

// Handle executes the functional Command Handler.
func (fn HandlerFunc) Handle(ctx context.Context, cmd eventually.Command) error {
	return fn(ctx, cmd)
}

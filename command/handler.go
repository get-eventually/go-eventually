package command

import (
	"context"

	"github.com/get-eventually/go-eventually"
)

// Command is a Message representing an action being performed by something
// or somebody.
//
// In order to enforce this concept, it is suggested to name Command types
// using "present tense".
type Command eventually.Message

// Type is a marker interface that represents the type of a Domain Command.
type Type eventually.Payload

// Handler is the interface that defines a Command Handler, component
// that receives a specific kind of Command and executes the business logic
// related to that particular Command.
type Handler interface {
	// CommandType returns an empty instance of the Type type the Handler
	// is supposed to handle.
	CommandType() Type

	// Handle receives a Domain Command and executes the necessary business logic.
	//
	// Since Commands' responsibility is only to trigger side effects, and not
	// answering Domain Queries, the Handle method only returns an error, whether
	// the Command has failed execution or not.
	Handle(context.Context, Command) error
}

var _ Handler = HandlerFunc(nil)

// HandlerFunc is a functional Command Handler.
type HandlerFunc func(context.Context, Command) error

// CommandType returns nil.
//
// Make sure not to use HandlerFunc in a command.Dispatcher, because it will
// probably require this function to return a meaningful value.
func (fn HandlerFunc) CommandType() Type {
	return nil
}

// Handle executes the functional Command Handler.
func (fn HandlerFunc) Handle(ctx context.Context, cmd Command) error {
	return fn(ctx, cmd)
}

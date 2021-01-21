package command

import (
	"context"
	"fmt"
	"reflect"

	"github.com/eventually-rs/eventually-go"
)

// Dispatcher represents a component that routes Domain Commands into
// their appropriate Command Handlers.
//
// Dispatchers might be synchronous, where the Command is directly sent to
// the Command Handler and the business logic is executed synchronously.
//
// Dispatchers might also be asynchronous, where the Command is send to an
// external queue and return from the Dispatch call immediately, while
// the Command Handler picks up the Command from the external queue
// and executes the business logic.
//
// Make sure to pick the right kind of Command Dispatcher that suits your
// application needs.
type Dispatcher interface {
	Dispatch(context.Context, eventually.Command) error
}

// SimpleBus is a simple Command Bus implementation that allows to register
// Command Handlers and dispatch Domain Commands.
//
// Use NewSimpleBus to create a new SimpleBus instance.
type SimpleBus struct {
	handlers map[reflect.Type]Handler
}

// NewSimpleBus returns a new instance of SimpleBus.
func NewSimpleBus() *SimpleBus {
	return &SimpleBus{handlers: make(map[reflect.Type]Handler)}
}

// Register adds the specified Command Handler into the SimpleBus routing table,
// so as to be able to start dispatching Commands to the Handler when
// calling Dispatch.
//
// Please note, when registering multiple Handlers accepting the same Command type,
// the last registration will overwrite any previous Handler registration, as
// this SimpleBus does not support voting.
func (b *SimpleBus) Register(handler Handler) {
	typ := reflect.TypeOf(handler.CommandType())
	b.handlers[typ] = handler
}

// Dispatch accepts incoming Domain Commands and route them to their
// appropriate Command Handlers, if registered.
//
// An error is returned if no Command Handler has been registered
// for the Domain Command submitted.
//
// Dispatching strategy for SimpleBus is synchronous, meaning that
// the Dispatch method returns when the Command Handler finished execution.
// If the Command Handler fails execution, returning an error, this error
// is also returned by the Dispatch invocation.
func (b SimpleBus) Dispatch(ctx context.Context, cmd eventually.Command) error {
	typ := reflect.TypeOf(cmd.Payload)

	handler, ok := b.handlers[typ]
	if !ok {
		return fmt.Errorf("command.SimpleBus: no handler registered for type %s", typ.Name())
	}

	if err := handler.Handle(ctx, cmd); err != nil {
		return fmt.Errorf("command.SimpleBus: failed to execute command: %w", err)
	}

	return nil
}

package command

import (
	"context"
	"fmt"
	"reflect"
	"sync"
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
	Dispatch(context.Context, Command) error
}

var _ Dispatcher = InMemoryDispatcher{}

// InMemoryDispatcher is a in-memory Command Dispatcher implementation that allows to register
// Command Handlers and dispatch Domain Commands.
//
// Use NewInMemoryDispatcher to create a new InMemoryDispatcher instance.
type InMemoryDispatcher struct {
	handlers map[reflect.Type]Handler
}

// NewInMemoryDispatcher returns a new instance of InMemoryDispatcher.
func NewInMemoryDispatcher() InMemoryDispatcher {
	return InMemoryDispatcher{handlers: make(map[reflect.Type]Handler)}
}

// Register adds the specified Command Handler into the InMemoryDispatcher routing table,
// so as to be able to start dispatching Commands to the Handler when
// calling Dispatch.
//
// Please note, when registering multiple Handlers accepting the same Command type,
// the last registration will overwrite any previous Handler registration, as
// this InMemoryDispatcher does not support voting.
func (d InMemoryDispatcher) Register(handler Handler) {
	typ := reflect.TypeOf(handler.CommandType())
	d.handlers[typ] = handler
}

// Dispatch accepts incoming Domain Commands and route them to their
// appropriate Command Handlers, if registered.
//
// An error is returned if no Command Handler has been registered
// for the Domain Command submitted.
//
// Dispatching strategy for InMemoryDispatcher is synchronous, meaning that
// the Dispatch method returns when the Command Handler finished execution.
// If the Command Handler fails execution, returning an error, this error
// is also returned by the Dispatch invocation.
func (d InMemoryDispatcher) Dispatch(ctx context.Context, cmd Command) error {
	typ := reflect.TypeOf(cmd.Payload)

	handler, ok := d.handlers[typ]
	if !ok {
		return fmt.Errorf("command.InMemoryDispatcher: no handler registered for type %s", typ.Name())
	}

	if err := handler.Handle(ctx, cmd); err != nil {
		return fmt.Errorf("command.InMemoryDispatcher: failed to execute command: %w", err)
	}

	return nil
}

var _ Dispatcher = &TrackingDispatcher{}

// TrackingDispatcher is a fake component that can be used as a command.Dispatcher
// instance, but keeps the received command in-memory.
//
// Useful for testing, this implementation is thread-safe.
type TrackingDispatcher struct {
	mx               sync.RWMutex
	recordedCommands []Command
}

// NewTrackingDispatcher creates a new instance of a fake in-memory command.Dispatcher.
func NewTrackingDispatcher() *TrackingDispatcher {
	return new(TrackingDispatcher)
}

// Dispatch records the provided Command internally.
func (cd *TrackingDispatcher) Dispatch(ctx context.Context, cmd Command) error {
	cd.mx.Lock()
	defer cd.mx.Unlock()

	cd.recordedCommands = append(cd.recordedCommands, cmd)

	return nil
}

// RecordedCommands returns the list of Commands recorded by the dispatcher.
func (cd *TrackingDispatcher) RecordedCommands() []Command {
	cd.mx.RLock()
	defer cd.mx.RUnlock()

	return cd.recordedCommands
}

// FlushCommands returns the list of Commands recorded by the dispatcher and
// resets the internal list to nil.
func (cd *TrackingDispatcher) FlushCommands() []Command {
	cd.mx.Lock()
	defer cd.mx.Unlock()

	commands := cd.recordedCommands
	cd.recordedCommands = nil

	return commands
}

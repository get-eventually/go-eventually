package internal

import (
	"context"
	"sync"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/command"
)

var _ command.Dispatcher = &InMemoryCommandDispatcher{}

// InMemoryCommandDispatcher is a fake component that can be used as a command.Dispatcher
// instance, but keeps the received command in-memory.
//
// Useful for testing, this implementation is thread-safe.
type InMemoryCommandDispatcher struct {
	mx               sync.RWMutex
	recordedCommands []eventually.Command
}

// NewInMemoryCommandDispatcher creates a new instance of a fake in-memory command.Dispatcher.
func NewInMemoryCommandDispatcher() *InMemoryCommandDispatcher {
	return new(InMemoryCommandDispatcher)
}

// Dispatch records the provided Command internally.
func (cd *InMemoryCommandDispatcher) Dispatch(ctx context.Context, cmd eventually.Command) error {
	cd.mx.Lock()
	defer cd.mx.Unlock()

	cd.recordedCommands = append(cd.recordedCommands, cmd)

	return nil
}

// RecordedCommands returns the list of Commands recorded by the dispatcher.
func (cd *InMemoryCommandDispatcher) RecordedCommands() []eventually.Command {
	cd.mx.RLock()
	defer cd.mx.RUnlock()

	return cd.recordedCommands
}

// FlushCommands returns the list of Commands recorded by the dispatcher and
// resets the internal list to nil.
func (cd *InMemoryCommandDispatcher) FlushCommands() []eventually.Command {
	cd.mx.Lock()
	defer cd.mx.Unlock()

	commands := cd.recordedCommands
	cd.recordedCommands = nil

	return commands
}

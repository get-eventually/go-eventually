package command

import (
	"context"
	"fmt"
	"reflect"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"
)

var (
	ErrCommandTypeNotRegistered = fmt.Errorf("command.EventStoredDispatcher: command type was not registered")
)

var _ Dispatcher = &EventStoredDispatcher{}

type EventStoredDispatcher struct {
	eventStore   eventstore.Typed
	typesToNames map[reflect.Type]string
	namesToTypes map[string]reflect.Type
}

func NewEventStoredDispatcher(
	ctx context.Context,
	es eventstore.Store,
	commands map[string]interface{},
) (*EventStoredDispatcher, error) {
	if err := es.Register(ctx, "command", commands); err != nil {
		return nil, fmt.Errorf("command.EventStoredDispatcher: failed to register command types: %w", err)
	}

	typed, err := es.Type(ctx, "command")
	if err != nil {
		return nil, fmt.Errorf("command.EventStoredDispatcher: failed to get typed event store access: %w", err)
	}

	dispatcher := EventStoredDispatcher{
		eventStore:   typed,
		typesToNames: make(map[reflect.Type]string, len(commands)),
		namesToTypes: make(map[string]reflect.Type, len(commands)),
	}

	for name, command := range commands {
		commandType := reflect.TypeOf(command)
		dispatcher.typesToNames[commandType] = name
		dispatcher.namesToTypes[name] = commandType
	}

	return &dispatcher, nil
}

func (esd *EventStoredDispatcher) Dispatch(ctx context.Context, command eventually.Command) error {
	commandType := reflect.TypeOf(command.Payload)
	commandName, ok := esd.typesToNames[commandType]

	if !ok {
		return ErrCommandTypeNotRegistered
	}

	_, err := esd.eventStore.
		Instance(commandName).
		Append(ctx, -1, eventually.Event(command))

	if err != nil {
		return fmt.Errorf("command.EventStoredDispatcher: failed to append new command to store: %w", err)
	}

	return nil
}

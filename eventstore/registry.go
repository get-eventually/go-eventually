package eventstore

import (
	"fmt"
	"reflect"

	"github.com/get-eventually/go-eventually"
)

// DeserializerFn is a function that deserializes a raw input into a Go type,
// which is passed here as interface{}, typically by reference.
type DeserializerFn func(msg []byte, v interface{}) error

// Registry contains type information about events to deserialize,
// and the deserialization function, when retrieving events from an Event Store.
//
// Given the current limitation of Go with generics, the only way to provide
// type information for deserialization is to use interfaces and reflection.
// This component uses the event type identifier and reflection to deserialize
// messages coming from the Event Store.
type Registry struct {
	deserializerFn  DeserializerFn
	eventNameToType map[string]reflect.Type
	eventTypeToName map[reflect.Type]string
}

// NewRegistry creates a new registry for deserializing event types, using
// the provided deserializer.
//
// An error is returned if the deserializer is nil.
func NewRegistry(deserializer DeserializerFn) (Registry, error) {
	if deserializer == nil {
		return Registry{}, fmt.Errorf("eventstore.Registry: invalid deserializer provided")
	}

	return Registry{
		deserializerFn:  deserializer,
		eventNameToType: make(map[string]reflect.Type),
		eventTypeToName: make(map[reflect.Type]string),
	}, nil
}

// Register adds the type information to this registry for all the provided Payload types.
//
// An error is returned if any of the provided events is nil, or if two different event types
// with the same type identifier (from the Payload.Name() method) have been provided.
func (r Registry) Register(events ...eventually.Payload) error {
	for _, event := range events {
		if event == nil {
			return fmt.Errorf("eventstore.Registry: expected event type, nil was provided instead")
		}

		eventName := event.Name()
		eventType := reflect.TypeOf(event)

		if registeredType, ok := r.eventNameToType[eventName]; ok {
			if registeredType == eventType {
				// Type is already registered and the new one is the same as the
				// one already registered, so we can continue with the other event types.
				continue
			}

			return fmt.Errorf(
				"eventstore.Registry: event '%s' has been already registered with a different type",
				eventName,
			)
		}

		r.eventNameToType[eventName] = eventType
		r.eventTypeToName[eventType] = eventName
	}

	return nil
}

// Deserialize attempts to deserialize a raw message with the type referenced by the
// supplied event type identifier.
func (r Registry) Deserialize(eventType string, payload []byte) (eventually.Payload, error) {
	payloadType, ok := r.eventNameToType[eventType]
	if !ok {
		return nil, fmt.Errorf("eventstore.Registry: received unregistered event, '%s'", eventType)
	}

	vp := reflect.New(payloadType)
	if err := r.deserializerFn(payload, vp.Interface()); err != nil {
		return nil, fmt.Errorf("eventstore.Registry: failed to deserialize event: %w", err)
	}

	return vp.Elem().Interface().(eventually.Payload), nil
}

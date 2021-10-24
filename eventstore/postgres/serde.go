package postgres

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/get-eventually/go-eventually"
)

// Serializer can be used by the EventStore to delegate serialization
// of a Domain Event from the eventually.Payload format (domain object) to binary format.
type Serializer interface {
	Serialize(eventType string, event eventually.Payload) ([]byte, error)
}

// Deserializer can be used by the EventStore to delegate deserialization
// of a Domain Event from binary format to its corresponding Domain Object.
type Deserializer interface {
	Deserialize(eventType string, data []byte) (eventually.Payload, error)
}

// Serde is a serializer/deserializer type that can be used by the EventStore
// to serialize Domain Events to and deserialize Domain Events from the store.
type Serde interface {
	Serializer
	Deserializer
}

// FusedSerde is a convenience type to fuse a Serializer and Deserializer
// in a Serde instance.
type FusedSerde struct {
	Serializer
	Deserializer
}

// SerializerFunc is a functional type that implements the Serializer interface.
type SerializerFunc func(eventType string, event eventually.Payload) ([]byte, error)

// Serialize delegates the function call to the inner function value.
func (fn SerializerFunc) Serialize(eventType string, event eventually.Payload) ([]byte, error) {
	return fn(eventType, event)
}

// DeserializerFunc is a functional type that implements the Deserializer interface.
type DeserializerFunc func(eventType string, data []byte) (eventually.Payload, error)

// Deserialize delegates the function call to the inner function value.
func (fn DeserializerFunc) Deserialize(eventType string, data []byte) (eventually.Payload, error) {
	return fn(eventType, data)
}

// JSONRegistry is a Serde type that serializes and deserializes
// into and from the JSON representation of eventually.Payload types registered.
//
// Given the current limitation of Go with generics, the only way to provide
// type information for deserialization is to use interfaces and reflection.
type JSONRegistry struct {
	eventNameToType map[string]reflect.Type
	eventTypeToName map[reflect.Type]string
}

// NewJSONRegistry creates a new registry for deserializing event types, using
// the provided deserializer.
func NewJSONRegistry() JSONRegistry {
	return JSONRegistry{
		eventNameToType: make(map[string]reflect.Type),
		eventTypeToName: make(map[reflect.Type]string),
	}
}

// Register adds the type information to this registry for all the provided Payload types.
//
// An error is returned if any of the provided events is nil, or if two different event types
// with the same type identifier (from the Payload.Name() method) have been provided.
func (r JSONRegistry) Register(events ...eventually.Payload) error {
	for _, event := range events {
		if event == nil {
			return fmt.Errorf("postgres.Registry: expected event type, nil was provided instead")
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
				"postgres.Registry: event '%s' has been already registered with a different type",
				eventName,
			)
		}

		r.eventNameToType[eventName] = eventType
		r.eventTypeToName[eventType] = eventName
	}

	return nil
}

// Serialize serializes a Domain Event using its JSON representation.
func (r JSONRegistry) Serialize(eventType string, event eventually.Payload) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("postgres.Registry: failed to serialize: %w", err)
	}

	return data, nil
}

// Deserialize attempts to deserialize a raw message with the type referenced by the
// supplied event type identifier.
func (r JSONRegistry) Deserialize(eventType string, data []byte) (eventually.Payload, error) {
	payloadType, ok := r.eventNameToType[eventType]
	if !ok {
		return nil, fmt.Errorf("postgres.Registry: received unregistered event, '%s'", eventType)
	}

	vp := reflect.New(payloadType)
	if err := json.Unmarshal(data, vp.Interface()); err != nil {
		return nil, fmt.Errorf("postgres.Registry: failed to deserialize event: %w", err)
	}

	return vp.Elem().Interface().(eventually.Payload), nil
}

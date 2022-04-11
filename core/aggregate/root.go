package aggregate

import (
	"fmt"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"
)

// ID represents an Aggregate ID type.
//
// Aggregate IDs should be able to be marshaled into a string format,
// in order to be saved onto a named Event Stream.
type ID interface {
	fmt.Stringer
}

// Aggregate is the segregated interface, part of the Aggregate Root interface,
// that describes the left-folding behavior of Domain Events to update the
// Aggregate Root state.
type Aggregate interface {
	// Apply applies the specified Event to the Aggregate Root,
	// by causing a state change in the Aggregate Root instance.
	//
	// Since this method cause a state change, implementors should make sure
	// to use pointer semantics on their Aggregate Root method receivers.
	//
	// Please note, this method should not perform any kind of external request
	// and should be, save for the Aggregate Root state mutation, free of side effects.
	// For this reason, this method does not include a context.Context instance
	// in the input parameters.
	Apply(event.Event) error
}

type Internal interface {
	FlushRecordedEvents() []event.Envelope
}

// Root is the interface describing an Aggregate Root instance.
//
// This interface should be implemented by your Aggregate Root types.
// Make sure your Aggregate Root types embed the aggregate.BaseRoot type
// to complete the implementation of this interface.
type Root[I ID] interface {
	Aggregate
	Internal

	// AggregateID returns the Aggregate Root identifier.
	AggregateID() I

	// Version returns the current Aggregate Root version.
	// The version gets updated each time a new event is recorded
	// through the aggregate.RecordThat function.
	Version() version.Version

	setVersion(version.Version)
	recordThat(Aggregate, ...event.Envelope) error
}

// Type represents the type of an Aggregate, which will expose the
// name of the Aggregate (used as Event Store type).
//
// If your Aggregate implementation uses pointers, use the factory to
// return a non-nil instance of the type.
type Type[I ID, T Root[I]] struct {
	Name    string
	Factory func() T
}

// RecordThat records the Domain Event for the specified Aggregate Root.
//
// An error is typically returned if applying the Domain Event on the Aggregate
// Root instance fails with an error.
func RecordThat[I ID](root Root[I], event ...event.Envelope) error {
	return root.recordThat(root, event...)
}

// BaseRoot segregates and completes the aggregate.Root interface implementation
// when embedded to a user-defined Aggregate Root type.
//
// BaseRoot provides some common traits, such as tracking the current Aggregate
// Root version, and the recorded-but-uncommitted Domain Events, through
// the aggregate.RecordThat function.
type BaseRoot struct {
	version        version.Version
	recordedEvents []event.Envelope
}

// Version returns the current version of the Aggregate Root instance.
func (br BaseRoot) Version() version.Version { return br.version }

func (br *BaseRoot) FlushRecordedEvents() []event.Envelope {
	flushed := br.recordedEvents
	br.recordedEvents = nil

	return flushed
}

func (br *BaseRoot) setVersion(v version.Version) {
	br.version = v
}

func (br *BaseRoot) recordThat(aggregate Aggregate, events ...event.Envelope) error {
	for _, event := range events {
		if err := aggregate.Apply(event.Message); err != nil {
			return fmt.Errorf("%T: failed to record event: %w", br, err)
		}

		br.recordedEvents = append(br.recordedEvents, event)
		br.version++
	}

	return nil
}

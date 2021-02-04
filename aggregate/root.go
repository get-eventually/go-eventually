package aggregate

import (
	"fmt"

	"github.com/eventually-rs/eventually-go"
)

// ID represents an Aggregate ID type.
//
// Aggregate IDs should be able to be marshaled into a string format,
// in order to be saved onto a named Event Stream.
type ID interface {
	fmt.Stringer
}

// StringID is a string-typed Aggregate ID.
type StringID string

func (id StringID) String() string { return string(id) }

// Applier is the segregated interface, part of the Aggregate Root interface,
// that describes the left-folding behavior of Domain Events to update the
// Aggregate Root state.
type Applier interface {
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
	Apply(eventually.Event) error
}

// Root is the interface describing an Aggregate Root instance.
//
// This interface should be implemented by your Aggregate Root types.
// Make sure your Aggregate Root types embed the aggregate.BaseRoot type
// to complete the implementation of this interface.
type Root interface {
	Applier

	AggregateID() ID
	Version() int64

	updateVersion(int64)
	flushRecordedEvents() []eventually.Event
	recordThat(Applier, ...eventually.Event) error
}

// RecordThat records the Domain Event for the specified Aggregate Root.
//
// An error is typically returned if applying the Domain Event on the Aggregate
// Root instance fails with an error.
func RecordThat(root Root, event eventually.Event) error {
	return root.recordThat(root, event)
}

// BaseRoot segregates and completes the aggregate.Root interface implementation
// when embedded to a user-defined Aggregate Root type.
//
// BaseRoot provides some common traits, such as tracking the current Aggregate
// Root version, and the recorded-but-uncommitted Domain Events, through
// the aggregate.RecordThat function.
type BaseRoot struct {
	version        int64
	recordedEvents []eventually.Event
}

// Version returns the current version of the Aggregate Root instance.
func (br BaseRoot) Version() int64 { return br.version }

func (br *BaseRoot) updateVersion(v int64) { br.version = v }

func (br *BaseRoot) recordThat(aggregate Applier, events ...eventually.Event) error {
	for _, event := range events {
		if err := aggregate.Apply(event); err != nil {
			return fmt.Errorf("aggregate: failed to record event: %w", err)
		}

		br.recordedEvents = append(br.recordedEvents, event)
		br.version++
	}

	return nil
}

func (br *BaseRoot) flushRecordedEvents() []eventually.Event {
	flushed := br.recordedEvents
	br.recordedEvents = nil

	return flushed
}

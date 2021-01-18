package aggregate

import (
	"fmt"

	"github.com/eventually-rs/eventually-go"
)

type ID interface {
	fmt.Stringer
}

type StringID string

func (id StringID) String() string { return string(id) }

type Applier interface {
	Apply(eventually.Event) error
}

type Root interface {
	Applier

	AggregateID() ID
	Version() int64

	updateVersion(int64)
	flushRecordedEvents() []eventually.Event
	recordThat(Applier, ...eventually.Event) error
}

func RecordThat(root Root, event eventually.Event) error {
	return root.recordThat(root, event)
}

type BaseRoot struct {
	version        int64
	recordedEvents []eventually.Event `json:"-"`
}

func (br BaseRoot) Version() int64 { return br.version }

func (br *BaseRoot) updateVersion(v int64) { br.version = v }

func (br *BaseRoot) recordThat(aggregate Applier, events ...eventually.Event) error {
	for _, event := range events {
		if err := aggregate.Apply(event); err != nil {
			return fmt.Errorf("aggregate: failed to record event: %w", err)
		}

		br.recordedEvents = append(br.recordedEvents, event)
	}

	return nil
}

func (br *BaseRoot) flushRecordedEvents() []eventually.Event {
	flushed := br.recordedEvents
	br.recordedEvents = nil
	return flushed
}

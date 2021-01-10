package aggregate

import (
	"fmt"

	"github.com/eventually-rs/eventually-go"
)

type Root interface {
	eventually.EventApplier

	AggregateID() string
	Version() int64

	updateVersion(int64)
	flushRecordedEvents() []eventually.Event
	recordThat(eventually.EventApplier, ...eventually.Event) error
}

func Record(root Root, event interface{}) error {
	return RecordWithMetadata(root, event, nil)
}

func RecordWithMetadata(root Root, event interface{}, metadata eventually.Metadata) error {
	return root.recordThat(root, eventually.Event{
		Payload:  event,
		Metadata: metadata,
	})
}

type BaseRoot struct {
	version        int64
	recordedEvents []eventually.Event `json:"-"`
}

func (br BaseRoot) Version() int64 { return br.version }

func (br *BaseRoot) updateVersion(v int64) { br.version = v }

func (br *BaseRoot) recordThat(aggregate eventually.EventApplier, events ...eventually.Event) error {
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

package event

import (
	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/event/stream"
)

type Event eventually.Message

type Persisted struct {
	Event

	Stream         stream.ID
	Version        uint64
	SequenceNumber uint64
}

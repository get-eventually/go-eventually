package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/event"
)

var _ event.Event = new(Event)

type Event struct {
	ID         uuid.UUID
	RecordTime time.Time
	Kind       eventKind
}

// Name implements event.Event.
func (evt *Event) Name() string { return evt.Kind.Name() }

type eventKind interface {
	event.Event
	isEventKind()
}

var (
	_ eventKind = new(WasCreated)
	_ eventKind = new(EmailWasUpdated)
)

// WasCreated is the domain event fired after a User is created.
type WasCreated struct {
	FirstName string
	LastName  string
	BirthDate time.Time
	Email     string
}

// Name implements message.Message.
func (*WasCreated) Name() string { return "UserWasCreated" }
func (*WasCreated) isEventKind() {}

// EmailWasUpdated is the domain event fired after a User email is updated.
type EmailWasUpdated struct {
	Email string
}

// Name implements message.Message.
func (*EmailWasUpdated) Name() string { return "UserEmailWasUpdated" }
func (*EmailWasUpdated) isEventKind() {}

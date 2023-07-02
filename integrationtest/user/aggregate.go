// Package user serves as a small domain example of how to model
// an Aggregate using go-eventually.
//
// This package is used for integration tests in the parent module.
package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
)

// Type is the User aggregate type.
var Type = aggregate.Type[uuid.UUID, *User]{
	Name:    "User",
	Factory: func() *User { return &User{} },
}

// WasCreated is the domain event fired after a User is created.
type WasCreated struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	BirthDate time.Time
	Email     string
}

// Name implements message.Message.
func (WasCreated) Name() string {
	return "UserWasCreated"
}

// EmailWasUpdated is the domain event fired after a User email is updated.
type EmailWasUpdated struct {
	Email string
}

// Name implements message.Message.
func (EmailWasUpdated) Name() string {
	return "UserEmailWasUpdated"
}

var _ aggregate.Root[uuid.UUID] = &User{}

// User is a naive user implementation, modeled as an Aggregate
// using go-eventually's API.
type User struct {
	aggregate.BaseRoot

	// Aggregate field should remain unexported if possible,
	// to enforce encapsulation.

	id        uuid.UUID
	firstName string
	lastName  string
	birthDate time.Time
	email     string
}

// Apply implements aggregate.Aggregate.
func (user *User) Apply(evt event.Event) error {
	switch evt := evt.(type) {
	case WasCreated:
		user.id = evt.ID
		user.firstName = evt.FirstName
		user.lastName = evt.LastName
		user.birthDate = evt.BirthDate
		user.email = evt.Email

	case EmailWasUpdated:
		user.email = evt.Email

	default:
		return fmt.Errorf("%T: unexpected event type, %T", user, evt)
	}

	return nil
}

// AggregateID implements aggregate.Root.
func (user *User) AggregateID() uuid.UUID {
	return user.id
}

// Create creates a new User using the provided input.
func Create(id uuid.UUID, firstName, lastName, email string, birthDate time.Time) (*User, error) {
	user := &User{}

	if firstName == "" {
		return nil, fmt.Errorf("%T: invalid first name, is empty", user)
	}

	if lastName == "" {
		return nil, fmt.Errorf("%T: invalid last name, is empty", user)
	}

	if email == "" {
		return nil, fmt.Errorf("%T: invalid email, is empty", user)
	}

	if birthDate.IsZero() {
		return nil, fmt.Errorf("%T: invalid birth date, not specified", user)
	}

	if err := aggregate.RecordThat[uuid.UUID](user, event.Envelope{
		Message: WasCreated{
			ID:        id,
			FirstName: firstName,
			LastName:  lastName,
			BirthDate: birthDate,
			Email:     email,
		},
	}); err != nil {
		return nil, fmt.Errorf("%T: failed to record domain event, %w", user, err)
	}

	return user, nil
}

// UpdateEmail updates the User email with the specified one.
func (user *User) UpdateEmail(email string, metadata message.Metadata) error {
	if email == "" {
		return fmt.Errorf("%T: invalid email, is empty", user)
	}

	if err := aggregate.RecordThat[uuid.UUID](user, event.Envelope{
		Message: EmailWasUpdated{
			Email: email,
		},
		Metadata: metadata,
	}); err != nil {
		return fmt.Errorf("%T: failed to record domain event, %w", user, err)
	}

	return nil
}

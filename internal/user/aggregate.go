// Package user serves as a small domain example of how to model
// an Aggregate using go-eventually.
//
// This package is used for integration tests in the parent module.
package user

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
)

// Type is the User aggregate type.
var Type = aggregate.Type[uuid.UUID, *Event, *User]{
	Name:    "User",
	Factory: func() *User { return new(User) },
}

// User is a naive user implementation, modeled as an Aggregate
// using go-eventually's API.
type User struct {
	aggregate.BaseRoot[*Event]

	// Aggregate field should remain unexported if possible,
	// to enforce encapsulation.

	id        uuid.UUID
	firstName string
	lastName  string
	birthDate time.Time
	email     string
}

// Apply implements aggregate.Aggregate.
func (user *User) Apply(evt *Event) error {
	switch kind := evt.Kind.(type) {
	case *WasCreated:
		user.id = evt.ID
		user.firstName = kind.FirstName
		user.lastName = kind.LastName
		user.birthDate = kind.BirthDate
		user.email = kind.Email
	case *EmailWasUpdated:
		user.email = kind.Email
	default:
		return fmt.Errorf("user.Apply: unexpected event type, %T", user, evt)
	}

	return nil
}

// AggregateID implements aggregate.Root.
func (user *User) AggregateID() uuid.UUID {
	return user.id
}

// All the errors returned by User methods.
var (
	ErrInvalidFirstName = errors.New("user: invalid first name, is empty")
	ErrInvalidLastName  = errors.New("user: invalid last name, is empty")
	ErrInvalidEmail     = errors.New("user: invalid email name, is empty")
	ErrInvalidBirthDate = errors.New("user: invalid birthdate, is empty")
)

// Create creates a new User using the provided input.
func Create(id uuid.UUID, firstName, lastName, email string, birthDate, now time.Time) (*User, error) {
	if firstName == "" {
		return nil, ErrInvalidFirstName
	}

	if lastName == "" {
		return nil, ErrInvalidLastName
	}

	if email == "" {
		return nil, ErrInvalidEmail
	}

	if birthDate.IsZero() {
		return nil, ErrInvalidBirthDate
	}

	user := new(User)

	if err := aggregate.RecordThat(user, event.ToEnvelope(&Event{
		ID:         id,
		RecordTime: now,
		Kind: &WasCreated{
			FirstName: firstName,
			LastName:  lastName,
			BirthDate: birthDate,
			Email:     email,
		},
	})); err != nil {
		return nil, fmt.Errorf("user.Create: failed to record domain event, %w", err)
	}

	return user, nil
}

// UpdateEmail updates the User email with the specified one.
func (user *User) UpdateEmail(email string, now time.Time, metadata message.Metadata) error {
	if email == "" {
		return ErrInvalidEmail
	}

	if err := aggregate.RecordThat(user, event.Envelope[*Event]{
		Metadata: metadata,
		Message: &Event{
			ID:         user.id,
			RecordTime: now,
			Kind:       &EmailWasUpdated{Email: email},
		},
	}); err != nil {
		return fmt.Errorf("%T: failed to record domain event, %w", user, err)
	}

	return nil
}

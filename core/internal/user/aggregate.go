package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
)

var Type = aggregate.Type[uuid.UUID, *User]{
	Name:    "User",
	Factory: func() *User { return &User{} },
}

type WasCreated struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	BirthDate time.Time
	Email     string
}

func (WasCreated) Name() string {
	return "UserWasCreated"
}

type EmailWasUpdated struct {
	Email string
}

func (EmailWasUpdated) Name() string {
	return "UserEmailWasUpdated"
}

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

func (user *User) Apply(event event.Event) error {
	switch evt := event.(type) {
	case WasCreated:
		user.id = evt.ID
		user.firstName = evt.FirstName
		user.lastName = evt.LastName
		user.birthDate = evt.BirthDate
		user.email = evt.Email

	case EmailWasUpdated:
		user.email = evt.Email

	default:
		return fmt.Errorf("%T: unexpected event type, %T", user, event)
	}

	return nil
}

func (user *User) AggregateID() uuid.UUID {
	return user.id
}

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

func (user *User) UpdateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("%T: invalid email, is empty", user)
	}

	if err := aggregate.RecordThat[uuid.UUID](user, event.Envelope{
		Message: EmailWasUpdated{
			Email: email,
		},
	}); err != nil {
		return fmt.Errorf("%T: failed to record domain event, %w", user, err)
	}

	return nil
}

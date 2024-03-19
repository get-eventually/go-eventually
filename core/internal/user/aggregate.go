package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
)

var (
	ErrInvalidFirstName = fmt.Errorf("user.User: invalid first name, is empty")
	ErrInvalidLastName  = fmt.Errorf("user.User: invalid last name, is empty")
	ErrInvalidEmail     = fmt.Errorf("user.User: invalid email, is empty")
	ErrInvalidBirthDate = fmt.Errorf("user.User: invalid birth date, not specified")
)

var Type = aggregate.Type[uuid.UUID, event.Event, *User]{
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
	aggregate.BaseRoot[event.Event]

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

	if err := aggregate.RecordThat[uuid.UUID, event.Event](user, event.Envelope[event.Event]{
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

	if err := aggregate.RecordThat[uuid.UUID, event.Event](user, event.Envelope[event.Event]{
		Message: EmailWasUpdated{
			Email: email,
		},
	}); err != nil {
		return fmt.Errorf("%T: failed to record domain event, %w", user, err)
	}

	return nil
}

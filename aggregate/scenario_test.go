package aggregate_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/internal/user"
)

func TestScenario(t *testing.T) {
	var (
		id        = uuid.New()
		firstName = "John"
		lastName  = "Ross"
		birthDate = time.Date(1990, 1, 1, 0, 0, 0, 0, time.Local)
		email     = "john@ross.com"
		now       = time.Now()
	)

	t.Run("test an aggregate function with one factory", func(t *testing.T) {
		aggregate.
			Scenario(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, firstName, lastName, email, birthDate, now)
			}).
			Then(1, event.ToEnvelope(&user.Event{
				ID:         id,
				RecordTime: now,
				Kind: &user.WasCreated{
					FirstName: firstName,
					LastName:  lastName,
					BirthDate: birthDate,
					Email:     email,
				},
			})).
			AssertOn(t)
	})

	t.Run("test an aggregate function with one factory call that returns an error", func(t *testing.T) {
		aggregate.
			Scenario(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, "", lastName, email, birthDate, now)
			}).
			ThenFails().
			AssertOn(t)
	})

	t.Run("test an aggregate function with one factory call that returns a specific error", func(t *testing.T) {
		aggregate.
			Scenario(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, "", lastName, email, birthDate, now)
			}).
			ThenError(user.ErrInvalidFirstName).
			AssertOn(t)
	})

	t.Run("test an aggregate function with one factory call that returns multiple errors with errors.Join()", func(t *testing.T) { //nolint:lll // It's ok in a test.
		aggregate.
			Scenario(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, "", "", "", time.Time{}, now)
			}).
			ThenErrors(
				user.ErrInvalidFirstName,
				user.ErrInvalidLastName,
				user.ErrInvalidEmail,
				user.ErrInvalidBirthDate,
			).
			AssertOn(t)
	})

	t.Run("test an aggregate function with an already-existing AggregateRoot instance", func(t *testing.T) {
		aggregate.
			Scenario(user.Type).
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(&user.Event{
					ID:         id,
					RecordTime: now,
					Kind: &user.WasCreated{
						FirstName: firstName,
						LastName:  lastName,
						BirthDate: birthDate,
						Email:     email,
					},
				}),
			}).
			When(func(u *user.User) error {
				return u.UpdateEmail("john.ross@email.com", now, nil)
			}).
			Then(2, event.ToEnvelope(&user.Event{
				ID:         id,
				RecordTime: now,
				Kind: &user.EmailWasUpdated{
					Email: "john.ross@email.com",
				},
			})).
			AssertOn(t)
	})
}

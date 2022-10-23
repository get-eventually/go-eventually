package scenario_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/internal/user"
	"github.com/get-eventually/go-eventually/core/test/scenario"
)

func TestAggregateRoot(t *testing.T) {
	var (
		id        = uuid.New()
		firstName = "John"
		lastName  = "Ross"
		birthDate = time.Date(1990, 1, 1, 0, 0, 0, 0, time.Local)
		email     = "john@ross.com"
	)

	t.Run("test an aggregate function with one factory", func(t *testing.T) {
		scenario.
			AggregateRoot(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, firstName, lastName, email, birthDate)
			}).
			Then(1, event.ToEnvelope(user.WasCreated{
				ID:        id,
				FirstName: firstName,
				LastName:  lastName,
				BirthDate: birthDate,
				Email:     email,
			})).
			AssertOn(t)
	})

	t.Run("test an aggregate function with one factory call that returns an error", func(t *testing.T) {
		scenario.
			AggregateRoot(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, "", lastName, email, birthDate)
			}).
			ThenFails().
			AssertOn(t)
	})

	t.Run("test an aggregate function with one factory call that returns a specific error", func(t *testing.T) {
		scenario.
			AggregateRoot(user.Type).
			When(func() (*user.User, error) {
				return user.Create(id, "", lastName, email, birthDate)
			}).
			ThenError(user.ErrInvalidFirstName).
			AssertOn(t)
	})

	t.Run("test an aggregate function with an already-existing AggregateRoot instance", func(t *testing.T) {
		scenario.
			AggregateRoot(user.Type).
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(user.WasCreated{
					ID:        id,
					FirstName: firstName,
					LastName:  lastName,
					BirthDate: birthDate,
					Email:     email,
				}),
			}).
			When(func(u *user.User) error {
				return u.UpdateEmail("john.ross@email.com")
			}).
			Then(2, event.ToEnvelope(user.EmailWasUpdated{
				Email: "john.ross@email.com",
			})).
			AssertOn(t)
	})
}

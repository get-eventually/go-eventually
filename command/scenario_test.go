package command_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/command"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/internal/user"
	"github.com/get-eventually/go-eventually/version"
)

func TestScenario(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	makeCommandHandler := func(s event.Store) user.CreateCommandHandler {
		return user.CreateCommandHandler{
			Clock:          func() time.Time { return now },
			UUIDGenerator:  func() uuid.UUID { return id },
			UserRepository: aggregate.NewEventSourcedRepository(s, user.Type),
		}
	}

	t.Run("fails when the given arguments are invalid", func(t *testing.T) {
		command.
			Scenario[user.CreateCommand, user.CreateCommandHandler]().
			When(command.Envelope[user.CreateCommand]{
				Message: user.CreateCommand{
					FirstName: "",
					LastName:  "",
					BirthDate: time.Time{},
					Email:     "",
				},
				Metadata: nil,
			}).
			ThenErrors(
				user.ErrInvalidFirstName,
				user.ErrInvalidLastName,
				user.ErrInvalidEmail,
				user.ErrInvalidBirthDate,
			).
			AssertOn(t, makeCommandHandler)
	})

	t.Run("create new user", func(t *testing.T) {
		command.
			Scenario[user.CreateCommand, user.CreateCommandHandler]().
			When(command.Envelope[user.CreateCommand]{
				Message: user.CreateCommand{
					FirstName: "John",
					LastName:  "Doe",
					BirthDate: now,
					Email:     "john@doe.com",
				},
				Metadata: nil,
			}).
			Then(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(&user.Event{
					ID:         id,
					RecordTime: now,
					Kind: &user.WasCreated{
						FirstName: "John",
						LastName:  "Doe",
						BirthDate: now,
						Email:     "john@doe.com",
					},
				}),
			}).
			AssertOn(t, makeCommandHandler)
	})

	t.Run("cannot create two duplicate users", func(t *testing.T) {
		command.
			Scenario[user.CreateCommand, user.CreateCommandHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(&user.Event{
					ID:         id,
					RecordTime: now,
					Kind: &user.WasCreated{
						FirstName: "John",
						LastName:  "Doe",
						BirthDate: now,
						Email:     "john@doe.com",
					},
				}),
			}).
			When(command.Envelope[user.CreateCommand]{
				Message: user.CreateCommand{
					FirstName: "John",
					LastName:  "Doe",
					BirthDate: now,
					Email:     "john@doe.com",
				},
				Metadata: nil,
			}).
			ThenError(version.ConflictError{
				Expected: 0,
				Actual:   1,
			}).
			AssertOn(t, makeCommandHandler)
	})
}

package scenario_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/internal/user"
	"github.com/get-eventually/go-eventually/core/test/scenario"
	"github.com/get-eventually/go-eventually/core/version"
)

func TestCommandHandler(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	t.Run("create new user", func(t *testing.T) {
		scenario.
			CommandHandler[user.CreateCommand, user.CreateCommandHandler]().
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
				Envelope: event.Envelope{
					Message: user.WasCreated{
						ID:        id,
						FirstName: "John",
						LastName:  "Doe",
						BirthDate: now,
						Email:     "john@doe.com",
					},
					Metadata: nil,
				},
			}).
			AssertOn(t, func(s event.Store) user.CreateCommandHandler {
				return user.CreateCommandHandler{
					UUIDGenerator: func() uuid.UUID {
						return id
					},
					UserRepository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})

	t.Run("cannot create two duplicate users", func(t *testing.T) {
		scenario.
			CommandHandler[user.CreateCommand, user.CreateCommandHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.Envelope{
					Message: user.WasCreated{
						ID:        id,
						FirstName: "John",
						LastName:  "Doe",
						BirthDate: now,
						Email:     "john@doe.com",
					},
					Metadata: nil,
				},
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
			AssertOn(t, func(s event.Store) user.CreateCommandHandler {
				return user.CreateCommandHandler{
					UUIDGenerator: func() uuid.UUID {
						return id
					},
					UserRepository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})
}

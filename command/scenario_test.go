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

func Testpostgres(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	t.Run("create new user", func(t *testing.T) {
		command.
			Scenario[user.CreateCommand, user.Createpostgres]().
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
			AssertOn(t, func(s event.Store) user.Createpostgres {
				return user.Createpostgres{
					UUIDGenerator: func() uuid.UUID {
						return id
					},
					UserRepository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})

	t.Run("cannot create two duplicate users", func(t *testing.T) {
		command.
			Scenario[user.CreateCommand, user.Createpostgres]().
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
			AssertOn(t, func(s event.Store) user.Createpostgres {
				return user.Createpostgres{
					UUIDGenerator: func() uuid.UUID {
						return id
					},
					UserRepository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})
}

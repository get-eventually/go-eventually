package scenario_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/internal/user"
	"github.com/get-eventually/go-eventually/core/query"
	"github.com/get-eventually/go-eventually/core/test/scenario"
)

func TestQueryHandler(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	userWasCreatedEvent := user.WasCreated{
		ID:        id,
		FirstName: "John",
		LastName:  "Doe",
		BirthDate: now,
		Email:     "john@doe.com",
	}

	t.Run("fails when using an invalid id value", func(t *testing.T) {
		scenario.
			QueryHandler[user.GetQuery, user.ViewModel, user.GetQueryHandler]().
			When(query.Envelope[user.GetQuery]{
				Message:  user.GetQuery{},
				Metadata: nil,
			}).
			ThenError(user.ErrEmptyID).
			AssertOn(t, func(s event.Store) user.GetQueryHandler {
				return user.GetQueryHandler{
					Repository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})

	t.Run("fails when requesting a user that doesn't exist", func(t *testing.T) {
		scenario.
			QueryHandler[user.GetQuery, user.ViewModel, user.GetQueryHandler]().
			When(query.Envelope[user.GetQuery]{
				Message: user.GetQuery{
					ID: id,
				},
				Metadata: nil,
			}).
			ThenError(aggregate.ErrRootNotFound).
			AssertOn(t, func(s event.Store) user.GetQueryHandler {
				return user.GetQueryHandler{
					Repository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})

	t.Run("returns an existing user", func(t *testing.T) {
		scenario.
			QueryHandler[user.GetQuery, user.ViewModel, user.GetQueryHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.Envelope{
					Message:  userWasCreatedEvent,
					Metadata: nil,
				},
			}).
			When(query.Envelope[user.GetQuery]{
				Message: user.GetQuery{
					ID: id,
				},
				Metadata: nil,
			}).
			Then(user.ViewModel{
				Version:   1,
				ID:        id,
				FirstName: userWasCreatedEvent.FirstName,
				LastName:  userWasCreatedEvent.LastName,
				BirthDate: userWasCreatedEvent.BirthDate,
				Email:     userWasCreatedEvent.Email,
			}).
			AssertOn(t, func(s event.Store) user.GetQueryHandler {
				return user.GetQueryHandler{
					Repository: aggregate.NewEventSourcedRepository(s, user.Type),
				}
			})
	})
}

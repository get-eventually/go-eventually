package query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/internal/user"
	"github.com/get-eventually/go-eventually/query"
)

func TestScenario(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	before := now.Add(-1 * time.Minute)

	expected := user.View{
		ID:        id,
		Version:   1,
		Email:     "me@email.com",
		FirstName: "John",
		LastName:  "Doe",
		BirthDate: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	makeQueryHandler := func(_ event.Store) *user.GetByEmailHandler {
		return user.NewGetByEmailHandler()
	}

	t.Run("returns the expected User by its email when it was just created", func(t *testing.T) {
		query.
			Scenario[user.GetByEmail, user.View, *user.GetByEmailHandler]().
			Given(event.Persisted{
				StreamID: event.StreamID(id.String()),
				Version:  1,
				Envelope: event.ToEnvelope(&user.Event{
					ID:         id,
					RecordTime: before,
					Kind: &user.WasCreated{
						FirstName: expected.FirstName,
						LastName:  expected.LastName,
						BirthDate: expected.BirthDate,
						Email:     expected.Email,
					},
				}),
			}).
			When(query.ToEnvelope(user.GetByEmail(expected.Email))).
			Then(expected).
			AssertOn(t, makeQueryHandler)
	})

	t.Run("returns user.ErrNotFound if the requested User does not exist", func(t *testing.T) {
		query.
			Scenario[user.GetByEmail, user.View, *user.GetByEmailHandler]().
			Given().
			When(query.ToEnvelope(user.GetByEmail(expected.Email))).
			ThenError(user.ErrNotFound).
			AssertOn(t, makeQueryHandler)
	})
}

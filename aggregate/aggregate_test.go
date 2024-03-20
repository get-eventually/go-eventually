package aggregate_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/internal/user"
)

func TestRoot(t *testing.T) {
	var (
		id        = uuid.New()
		firstName = "John"
		lastName  = "Doe"
		email     = "john@doe.com"
		birthDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		now       = time.Now()
	)

	t.Run("create new aggregate root", func(t *testing.T) {
		usr, err := user.Create(id, firstName, lastName, email, birthDate, now)
		assert.NoError(t, err)

		expectedEvents := event.ToEnvelopes(&user.Event{
			ID:         id,
			RecordTime: now,
			Kind: &user.WasCreated{
				FirstName: firstName,
				LastName:  lastName,
				BirthDate: birthDate,
				Email:     email,
			},
		})

		assert.Equal(t, expectedEvents, usr.FlushRecordedEvents())
	})

	t.Run("create new aggregate root with invalid fields", func(t *testing.T) {
		usr, err := user.Create(id, "", lastName, email, birthDate, now)
		assert.Error(t, err)
		assert.Nil(t, usr)
	})

	t.Run("update an existing aggregate root", func(t *testing.T) {
		usr, err := user.Create(id, firstName, lastName, email, birthDate, now)
		require.NoError(t, err)
		usr.FlushRecordedEvents() // NOTE: flushing previously-recorded events to simulate fetching from a repository.

		newEmail := "john.doe@email.com"

		err = usr.UpdateEmail(newEmail, now, nil)
		assert.NoError(t, err)

		expectedEvents := event.ToEnvelopes(&user.Event{
			ID:         id,
			RecordTime: now,
			Kind:       &user.EmailWasUpdated{Email: newEmail},
		})

		assert.Equal(t, expectedEvents, usr.FlushRecordedEvents())
	})
}

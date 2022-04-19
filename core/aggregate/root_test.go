package aggregate_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/internal/user"
)

func TestRoot(t *testing.T) {
	var (
		id        = uuid.New()
		firstName = "John"
		lastName  = "Doe"
		email     = "john@doe.com"
		birthDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("create new aggregate root", func(t *testing.T) {
		usr, err := user.Create(id, firstName, lastName, email, birthDate)
		assert.NoError(t, err)

		expectedEvents := []event.Envelope{
			{
				Message: user.WasCreated{
					ID:        id,
					FirstName: firstName,
					LastName:  lastName,
					BirthDate: birthDate,
					Email:     email,
				},
			},
		}

		assert.Equal(t, expectedEvents, usr.FlushRecordedEvents())
	})

	t.Run("create new aggregate root with invalid fields", func(t *testing.T) {
		usr, err := user.Create(id, "", lastName, email, birthDate)
		assert.Error(t, err)
		assert.Nil(t, usr)
	})

	t.Run("update an existing aggregate root", func(t *testing.T) {
		usr, err := user.Create(id, firstName, lastName, email, birthDate)
		require.NoError(t, err)
		usr.FlushRecordedEvents() // NOTE: flushing previously-recorded events to simulate fetching from a repository.

		newEmail := "john.doe@email.com"

		err = usr.UpdateEmail(newEmail)
		assert.NoError(t, err)

		expectedEvents := []event.Envelope{
			{
				Message: user.EmailWasUpdated{
					Email: newEmail,
				},
			},
		}

		assert.Equal(t, expectedEvents, usr.FlushRecordedEvents())
	})
}

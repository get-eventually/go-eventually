package aggregate_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/internal/user"
)

func TestEventSourcedRepository(t *testing.T) {
	var (
		id        = uuid.New()
		firstName = "John"
		lastName  = "Doe"
		email     = "john@doe.com"
		birthDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		now       = time.Now()
	)

	ctx := context.Background()
	eventStore := event.NewInMemoryStore()
	userRepository := aggregate.NewEventSourcedRepository(eventStore, user.Type)

	_, err := userRepository.Get(ctx, id)
	require.ErrorIs(t, err, aggregate.ErrRootNotFound)

	usr, err := user.Create(id, firstName, lastName, email, birthDate, now)
	require.NoError(t, err)

	err = userRepository.Save(ctx, usr)
	require.NoError(t, err)

	got, err := userRepository.Get(ctx, usr.AggregateID())
	require.NoError(t, err)
	assert.Equal(t, usr, got)
}

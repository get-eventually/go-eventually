package aggregate_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/internal/user"
	"github.com/get-eventually/go-eventually/core/test"
)

func TestEventSourcedRepository(t *testing.T) {
	var (
		id        = uuid.New()
		firstName = "John"
		lastName  = "Doe"
		email     = "john@doe.com"
		birthDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	ctx := context.Background()
	eventStore := test.NewInMemoryEventStore()
	userRepository := aggregate.NewEventSourcedRepository[uuid.UUID](
		eventStore,
		func() *user.User { return &user.User{} },
	)

	_, err := userRepository.Get(ctx, id)
	assert.ErrorIs(t, err, aggregate.ErrRootNotFound)

	usr, err := user.Create(id, firstName, lastName, email, birthDate)
	assert.NoError(t, err)

	err = userRepository.Save(ctx, usr)
	assert.NoError(t, err)

	got, err := userRepository.Get(ctx, usr.AggregateID())
	assert.NoError(t, err)
	assert.Equal(t, usr, got)
}
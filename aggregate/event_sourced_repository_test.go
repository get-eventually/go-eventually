package aggregate_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

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
	)

	ctx := context.Background()
	eventStore := event.NewInMemoryStore()
	userRepository := aggregate.NewEventSourcedRepository(eventStore, user.Type)

	_, err := userRepository.Get(ctx, id)
	if !assert.ErrorIs(t, err, aggregate.ErrRootNotFound) {
		return
	}

	usr, err := user.Create(id, firstName, lastName, email, birthDate, now)
	if !assert.NoError(t, err) {
		return
	}

	err = userRepository.Save(ctx, usr)
	if !assert.NoError(t, err) {
		return
	}

	got, err := userRepository.Get(ctx, usr.AggregateID())
	assert.NoError(t, err)
	assert.Equal(t, usr, got)
}

package integrationtest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/version"
	"github.com/get-eventually/go-eventually/integrationtest/user"
)

// AggregateRepository returns an executable testing suite running on the
// agfgregate.Repository value provided in input.
//
// The aggregate.Repository value requested should comply with the given signature.
//
// Package user of this module exposes a Protobuf-based serde, which can be useful
// to test serialization and deserialization of data to the target repository implementation.
func AggregateRepository(repository aggregate.Repository[uuid.UUID, *user.User]) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		t.Run("it can load and save aggregates from the database", func(t *testing.T) {
			var (
				id        = uuid.New()
				firstName = "John"
				lastName  = "Doe"
				email     = "john@doe.com"
				birthDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
			)

			_, err := repository.Get(ctx, id)
			if !assert.ErrorIs(t, err, aggregate.ErrRootNotFound) {
				return
			}

			usr, err := user.Create(id, firstName, lastName, email, birthDate)
			if !assert.NoError(t, err) {
				return
			}

			if err := repository.Save(ctx, usr); !assert.NoError(t, err) {
				return
			}

			got, err := repository.Get(ctx, id)
			assert.NoError(t, err)
			assert.Equal(t, usr, got)
		})

		t.Run("optimistic locking of aggregates is also working fine", func(t *testing.T) {
			var (
				id        = uuid.New()
				firstName = "John"
				lastName  = "Doe"
				email     = "john@doe.com"
				birthDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
			)

			usr, err := user.Create(id, firstName, lastName, email, birthDate)
			require.NoError(t, err)

			newEmail := "johndoe@gmail.com"
			require.NoError(t, usr.UpdateEmail(newEmail, message.Metadata{
				"Testing-Metadata-Time": time.Now().Format(time.RFC3339),
			}))

			if err := repository.Save(ctx, usr); !assert.NoError(t, err) {
				return
			}

			// Try to create a new User instance, but stop at Create.
			outdatedUsr, err := user.Create(id, firstName, lastName, email, birthDate)
			require.NoError(t, err)

			err = repository.Save(ctx, outdatedUsr)

			expectedErr := version.ConflictError{
				Expected: 0,
				Actual:   2,
			}

			var conflictErr version.ConflictError
			assert.ErrorAs(t, err, &conflictErr)
			assert.Equal(t, expectedErr, conflictErr)
		})
	}
}

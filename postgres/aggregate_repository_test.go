package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/version"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/postgres/internal/user"
	"github.com/get-eventually/go-eventually/postgres/internal/user/proto"
	"github.com/get-eventually/go-eventually/serdes"
)

const defaultPostgresURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

//nolint:lll // 121 characters are fine :)
func testUserRepository(t *testing.T) func(ctx context.Context, repository aggregate.Repository[uuid.UUID, *user.User]) {
	return func(ctx context.Context, repository aggregate.Repository[uuid.UUID, *user.User]) {
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
			require.NoError(t, usr.UpdateEmail(newEmail))

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

func TestAggregateRepository(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	url, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		url = defaultPostgresURL
	}

	require.NoError(t, postgres.RunMigrations(url))

	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, url)
	require.NoError(t, err)

	repository := postgres.AggregateRepository[uuid.UUID, *user.User]{
		Conn:          conn,
		AggregateType: user.Type,
		AggregateSerde: serdes.NewProtoJSON[*user.User, *proto.User](
			user.ProtoSerde,
			func() *proto.User { return &proto.User{} },
		),
		MessageSerde: serdes.NewProtoJSON[message.Message, *proto.Event](
			user.EventProtoSerde,
			func() *proto.Event { return &proto.Event{} },
		),
	}

	testUserRepository(t)(ctx, repository)
}

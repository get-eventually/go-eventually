package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/version"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/postgres/internal/user"
	"github.com/get-eventually/go-eventually/postgres/internal/user/proto"
)

const defaultPostgresURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

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
	conn, err := pgx.Connect(ctx, url)
	require.NoError(t, err)

	repository := postgres.AggregateRepository[uuid.UUID, *user.User]{
		Conn:              conn,
		AggregateTypeName: "User",
		// TODO(ar3s3ru): would be nice to expose a generic Protobuf serde wrapper.
		AggregateSerde: aggregate.Serde[uuid.UUID, *user.User, []byte]{
			Serializer: aggregate.SerializerFunc[uuid.UUID, *user.User, []byte](func(src *user.User) ([]byte, error) {
				model, err := user.ProtoSerde.Serialize(src)
				if err != nil {
					return nil, err
				}

				return protojson.Marshal(model)
			}),
			Deserializer: aggregate.DeserializerFunc[uuid.UUID, []byte, *user.User](func(src []byte) (*user.User, error) {
				model := &proto.User{}
				if err := protojson.Unmarshal(src, model); err != nil {
					return nil, err
				}

				return user.ProtoSerde.Deserialize(model)
			}),
		},
		MessageSerde: message.Serde[message.Message, []byte]{
			Serializer: message.SerializerFunc[message.Message, []byte](func(msg message.Message) ([]byte, error) {
				evt, err := user.EventProtoSerde.Serialize(msg)
				if err != nil {
					return nil, err
				}

				return protojson.Marshal(evt)
			}),
			Deserializer: message.DeserializerFunc[[]byte, message.Message](func(data []byte) (message.Message, error) {
				evt := &proto.Event{}
				if err := protojson.Unmarshal(data, evt); err != nil {
					return nil, err
				}

				return user.EventProtoSerde.Deserialize(evt)
			}),
		},
	}

	t.Run("it works", func(t *testing.T) {
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

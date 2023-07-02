package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/integrationtest"
	"github.com/get-eventually/go-eventually/integrationtest/user"
	"github.com/get-eventually/go-eventually/integrationtest/user/proto"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/serdes"
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
	conn, err := pgxpool.Connect(ctx, url)
	require.NoError(t, err)

	repository := postgres.AggregateRepository[uuid.UUID, *user.User]{
		Conn:          conn,
		AggregateType: user.Type,
		AggregateSerde: serdes.Chain[*user.User, *proto.User, []byte](
			user.ProtoSerde,
			serdes.NewProtoJSON(func() *proto.User { return &proto.User{} }),
		),
		MessageSerde: serdes.Chain[message.Message, *proto.Event, []byte](
			user.EventProtoSerde,
			serdes.NewProtoJSON(func() *proto.Event { return &proto.Event{} }),
		),
	}

	integrationtest.AggregateRepository(repository)(t)
}

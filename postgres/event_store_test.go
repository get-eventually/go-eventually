package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/integrationtest"
	"github.com/get-eventually/go-eventually/integrationtest/user"
	"github.com/get-eventually/go-eventually/integrationtest/user/proto"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/serdes"
)

func TestEventStore(t *testing.T) {
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

	eventStore := postgres.EventStore{
		Conn: conn,
		Serde: serdes.Chain[message.Message, *proto.Event, []byte](
			user.EventProtoSerde,
			serdes.NewProtoJSON(func() *proto.Event { return &proto.Event{} }),
		),
	}

	integrationtest.EventStore(eventStore)(t)
}

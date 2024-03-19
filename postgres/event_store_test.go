package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Used to bring in the driver for sql.Open.
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/internal/user"
	"github.com/get-eventually/go-eventually/internal/user/proto"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/serde"
)

func TestEventStore(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	url, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		url = defaultPostgresURL
	}

	db, err := sql.Open("pgx", url)
	require.NoError(t, err)
	require.NoError(t, postgres.RunMigrations(db))
	require.NoError(t, db.Close())

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, url)
	require.NoError(t, err)

	eventStore := postgres.EventStore{
		Conn: conn,
		Serde: serde.Chain[message.Message, *proto.Event, []byte](
			user.EventProtoSerde,
			serde.NewProtoJSON(func() *proto.Event { return new(proto.Event) }),
		),
	}

	user.EventStoreSuite(eventStore)(t)
}

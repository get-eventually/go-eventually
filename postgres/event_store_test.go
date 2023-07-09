package eventuallypostgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Used to bring in the driver for sql.Open.
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/integrationtest"
	"github.com/get-eventually/go-eventually/integrationtest/user"
	"github.com/get-eventually/go-eventually/integrationtest/user/proto"
	eventuallypostgres "github.com/get-eventually/go-eventually/postgres"
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

	db, err := sql.Open("pgx", url)
	require.NoError(t, err)
	require.NoError(t, eventuallypostgres.RunMigrations(db))
	require.NoError(t, db.Close())

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, url)
	require.NoError(t, err)

	eventStore := eventuallypostgres.EventStore{
		Conn: conn,
		Serde: serdes.Chain[message.Message, *proto.Event, []byte](
			user.EventProtoSerde,
			serdes.NewProtoJSON(func() *proto.Event { return &proto.Event{} }),
		),
	}

	integrationtest.EventStore(eventStore)(t)
}

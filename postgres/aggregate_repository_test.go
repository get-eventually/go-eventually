package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Used to bring in the driver for sql.Open.
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/internal/user"
	userv1 "github.com/get-eventually/go-eventually/internal/user/gen/user/v1"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/postgres/internal"
	"github.com/get-eventually/go-eventually/serde"
)

func TestAggregateRepository(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ctx := context.Background()

	container, err := internal.NewPostgresContainer(ctx)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, container.Terminate(ctx))
	}()

	db, err := sql.Open("pgx", container.ConnectionDSN)
	require.NoError(t, err)
	require.NoError(t, postgres.RunMigrations(db))
	require.NoError(t, db.Close())

	conn, err := pgxpool.New(ctx, container.ConnectionDSN)
	require.NoError(t, err)

	user.AggregateRepositorySuite(postgres.NewAggregateRepository(
		conn, user.Type,
		serde.Chain(
			user.ProtoSerde,
			serde.NewProtoJSON(func() *userv1.User { return new(userv1.User) }),
		),
		serde.Chain(
			user.EventProtoSerde,
			serde.NewProtoJSON(func() *userv1.Event { return new(userv1.Event) }),
		),
		postgres.WithAggregateTableName[uuid.UUID, *user.User](postgres.DefaultAggregateTableName),
		postgres.WithEventsTableName[uuid.UUID, *user.User](postgres.DefaultEventsTableName),
		postgres.WithStreamsTableName[uuid.UUID, *user.User](postgres.DefaultStreamsTableName),
	))(t)
}

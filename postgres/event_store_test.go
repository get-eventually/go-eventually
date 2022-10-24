package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/version"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/postgres/internal/user"
	"github.com/get-eventually/go-eventually/postgres/internal/user/proto"
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

	repository := aggregate.NewEventSourcedRepository(eventStore, user.Type)

	testUserRepository(t)(ctx, repository)

	t.Run("append works when used with version.CheckAny", func(t *testing.T) {
		id := uuid.New()

		usr, err := user.Create(id, "Dani", "Ross", "dani@ross.com", time.Now())
		require.NoError(t, err)

		require.NoError(t, usr.UpdateEmail("dani.ross@mail.com"))

		eventsToCommit := usr.FlushRecordedEvents()
		expectedVersion := version.Version(len(eventsToCommit))

		newVersion, err := eventStore.Append(
			ctx,
			event.StreamID(id.String()),
			version.Any,
			eventsToCommit...,
		)

		require.NoError(t, err)
		require.Equal(t, expectedVersion, newVersion)

		// Now let's update the User event stream once more.

		require.NoError(t, usr.UpdateEmail("daniross123@gmail.com"))

		newEventsToCommit := usr.FlushRecordedEvents()
		expectedVersion += version.Version(len(newEventsToCommit))

		newVersion, err = eventStore.Append(
			ctx,
			event.StreamID(id.String()),
			version.Any,
			newEventsToCommit...,
		)

		require.NoError(t, err)
		require.Equal(t, expectedVersion, newVersion)
	})
}

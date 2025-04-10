package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/version"
)

// EventStoreSuite returns an executable testing suite running on the event.Store
// value provided in input.
func EventStoreSuite(eventStore event.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()

		// Testing the Event-sourced repository implementation, which indirectly
		// tests the Event Store instance.
		AggregateRepositorySuite(aggregate.NewEventSourcedRepository(eventStore, Type))(t)

		t.Run("append works when used with version.CheckAny", func(t *testing.T) {
			id := uuid.New()

			usr, err := Create(id, "Dani", "Ross", "dani@ross.com", now, now)
			require.NoError(t, err)

			require.NoError(t, usr.UpdateEmail("dani.ross@mail.com", now, nil))

			eventsToCommit := usr.FlushRecordedEvents()
			expectedVersion := version.Version(len(eventsToCommit)) //nolint:gosec // This should not overflow.

			newVersion, err := eventStore.Append(
				ctx,
				event.StreamID(id.String()),
				version.Any,
				eventsToCommit...,
			)

			require.NoError(t, err)
			require.Equal(t, expectedVersion, newVersion)

			// Now let's update the User event stream once more.

			require.NoError(t, usr.UpdateEmail("daniross123@gmail.com", now, nil))

			newEventsToCommit := usr.FlushRecordedEvents()
			expectedVersion += version.Version(len(newEventsToCommit)) //nolint:gosec // This should not overflow.

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
}

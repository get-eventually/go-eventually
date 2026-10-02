package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Used to bring in the driver for sql.Open.
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/internal/user"
	userv1 "github.com/get-eventually/go-eventually/internal/user/gen/user/v1"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/postgres"
	"github.com/get-eventually/go-eventually/postgres/internal"
	"github.com/get-eventually/go-eventually/serde"
	"github.com/get-eventually/go-eventually/version"
)

var _ aggregate.Repository[uuid.UUID, *user.User] = postgres.TransactionAwareAggregateRepository[uuid.UUID, *user.User]{}

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

type transactionKey struct{}

func retrieveTransaction(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(transactionKey{}).(pgx.Tx)
	return tx, ok
}

func newTransactionAwareRepository(
	retrieve postgres.TxRetriever,
	options ...postgres.Option[uuid.UUID, *user.User],
) postgres.TransactionAwareAggregateRepository[uuid.UUID, *user.User] {
	return postgres.NewTransactionAwareAggregateRepository(
		retrieve, user.Type,
		serde.Chain(user.ProtoSerde, serde.NewProtoJSON(func() *userv1.User { return new(userv1.User) })),
		serde.Chain(user.EventProtoSerde, serde.NewProtoJSON(func() *userv1.Event { return new(userv1.Event) })),
		options...,
	)
}

func newTransactionTestUser(t *testing.T) *user.User {
	t.Helper()

	now := time.Now().UTC()
	birthDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	root, err := user.Create(uuid.New(), "John", "Doe", "john@doe.com", birthDate, now)
	require.NoError(t, err)

	return root
}

func TestTransactionAwareAggregateRepositoryRequiresTransaction(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		tx   pgx.Tx
		ok   bool
	}{
		{name: "missing"},
		{name: "nil transaction", ok: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repo := newTransactionAwareRepository(func(context.Context) (pgx.Tx, bool) {
				return test.tx, test.ok
			})
			root := newTransactionTestUser(t)
			ctx := t.Context()

			got, err := repo.Get(ctx, root.AggregateID())
			require.ErrorIs(t, err, postgres.ErrTransactionRequired)
			assert.Nil(t, got)
			require.ErrorIs(t, repo.Save(ctx, root), postgres.ErrTransactionRequired)
			assert.Len(t, root.FlushRecordedEvents(), 1)
		})
	}
}

type transactionSpy struct {
	pgx.Tx
	begins    int
	commits   int
	rollbacks int
}

func (tx *transactionSpy) Begin(ctx context.Context) (pgx.Tx, error) {
	tx.begins++
	return tx.Tx.Begin(ctx)
}

func (tx *transactionSpy) Commit(ctx context.Context) error {
	tx.commits++
	return tx.Tx.Commit(ctx)
}

func (tx *transactionSpy) Rollback(ctx context.Context) error {
	tx.rollbacks++
	return tx.Tx.Rollback(ctx)
}

func TestTransactionAwareAggregateRepository(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ctx := t.Context()
	container, err := internal.NewPostgresContainer(ctx)
	require.NoError(t, err)

	defer func() { require.NoError(t, container.Terminate(context.Background())) }()

	db, err := sql.Open("pgx", container.ConnectionDSN)
	require.NoError(t, err)
	require.NoError(t, postgres.RunMigrations(db))
	require.NoError(t, db.Close())

	pool, err := pgxpool.New(ctx, container.ConnectionDSN)
	require.NoError(t, err)

	defer pool.Close()

	_, err = pool.Exec(ctx, `
		CREATE SCHEMA custom;
		ALTER TABLE aggregates SET SCHEMA custom;
		ALTER TABLE event_streams SET SCHEMA custom;
		ALTER TABLE events SET SCHEMA custom;
		ALTER TABLE custom.aggregates ADD CONSTRAINT max_version CHECK (version < 2);
	`)
	require.NoError(t, err)

	repo := newTransactionAwareRepository(retrieveTransaction,
		postgres.WithAggregateTableName[uuid.UUID, *user.User]("custom.aggregates"),
		postgres.WithStreamsTableName[uuid.UUID, *user.User]("custom.event_streams"),
		postgres.WithEventsTableName[uuid.UUID, *user.User]("custom.events"),
	)

	for _, commit := range []bool{false, true} {
		name := "rollback"
		if commit {
			name = "commit"
		}

		t.Run(name, func(t *testing.T) {
			root := newTransactionTestUser(t)
			tx := beginRepositoryTransaction(ctx, t, pool)
			txCtx := context.WithValue(ctx, transactionKey{}, tx)
			_, err := repo.Get(txCtx, root.AggregateID())
			require.ErrorIs(t, err, aggregate.ErrRootNotFound)
			require.NoError(t, repo.Save(txCtx, root))
			got, err := repo.Get(txCtx, root.AggregateID())
			require.NoError(t, err)
			assert.Equal(t, root, got)
			assertRepositoryRows(txCtx, t, tx, root.AggregateID(), 1)
			assertRepositoryRows(ctx, t, pool, root.AggregateID(), 0)

			stale, err := user.Create(root.AggregateID(), "John", "Doe", "john@doe.com", time.Now(), time.Now())
			require.NoError(t, err)

			var conflict version.ConflictError
			require.ErrorAs(t, repo.Save(txCtx, stale), &conflict)
			assert.Zero(t, tx.begins)
			assert.Zero(t, tx.commits)
			assert.Zero(t, tx.rollbacks)

			if commit {
				require.NoError(t, tx.Commit(ctx))
				assertRepositoryRows(ctx, t, pool, root.AggregateID(), 1)
			} else {
				require.NoError(t, tx.Rollback(ctx))
				assertRepositoryRows(ctx, t, pool, root.AggregateID(), 0)
			}

			_, err = repo.Get(txCtx, root.AggregateID())
			require.ErrorIs(t, err, pgx.ErrTxClosed)
			require.ErrorIs(t, repo.Save(txCtx, newTransactionTestUser(t)), pgx.ErrTxClosed)
		})
	}

	t.Run("snapshot failure rolls back staged events", func(t *testing.T) {
		root := newTransactionTestUser(t)
		tx := beginRepositoryTransaction(ctx, t, pool)
		txCtx := context.WithValue(ctx, transactionKey{}, tx)
		require.NoError(t, repo.Save(txCtx, root))
		require.NoError(t, tx.Commit(ctx))

		tx = beginRepositoryTransaction(ctx, t, pool)
		txCtx = context.WithValue(ctx, transactionKey{}, tx)
		got, err := repo.Get(txCtx, root.AggregateID())
		require.NoError(t, err)
		require.NoError(t, got.UpdateEmail("updated@doe.com", time.Now(), message.Metadata{}))
		require.Error(t, repo.Save(txCtx, got))
		assert.Zero(t, tx.begins)
		assert.Zero(t, tx.commits)
		assert.Zero(t, tx.rollbacks)
		require.NoError(t, tx.Rollback(ctx))
		assertRepositoryRows(ctx, t, pool, root.AggregateID(), 1)

		tx = beginRepositoryTransaction(ctx, t, pool)
		got, err = repo.Get(context.WithValue(ctx, transactionKey{}, tx), root.AggregateID())
		require.NoError(t, err)
		assert.Equal(t, root, got)
		require.NoError(t, tx.Rollback(ctx))
	})
}

func beginRepositoryTransaction(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *transactionSpy {
	t.Helper()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := tx.Rollback(context.WithoutCancel(ctx))
		if err != nil {
			require.ErrorIs(t, err, pgx.ErrTxClosed)
		}
	})

	return &transactionSpy{Tx: tx}
}

type repositoryQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func assertRepositoryRows(ctx context.Context, t *testing.T, db repositoryQueryRower, id uuid.UUID, expected int) {
	t.Helper()

	var snapshots, streams, events int

	err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM custom.aggregates WHERE aggregate_id = $1),
			(SELECT count(*) FROM custom.event_streams WHERE event_stream_id = $1),
			(SELECT count(*) FROM custom.events WHERE event_stream_id = $1)
	`, id.String()).Scan(&snapshots, &streams, &events)
	require.NoError(t, err)
	assert.Equal(t, expected, snapshots)
	assert.Equal(t, expected, streams)
	assert.Equal(t, expected, events)
}

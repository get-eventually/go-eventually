package firestore_test

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/gcloud"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	eventuallyfirestore "github.com/get-eventually/go-eventually/firestore"
	"github.com/get-eventually/go-eventually/internal/user"
	userv1 "github.com/get-eventually/go-eventually/internal/user/gen/user/v1"
	"github.com/get-eventually/go-eventually/serde"
)

func TestEventStore(t *testing.T) {
	const projectID = "firestore-project"

	if testing.Short() {
		t.SkipNow()
	}

	ctx := context.Background()

	firestoreContainer, err := gcloud.RunFirestoreContainer(
		ctx,
		testcontainers.WithImage("google/cloud-sdk:469.0.0-emulators"),
		gcloud.WithProjectID(projectID),
	)
	require.NoError(t, err)

	// Clean up the container
	defer func() {
		require.NoError(t, firestoreContainer.Terminate(ctx))
	}()

	conn, err := grpc.Dial(firestoreContainer.URI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client, err := firestore.NewClient(ctx, projectID, option.WithGRPCConn(conn))
	require.NoError(t, err)

	defer func() {
		require.NoError(t, client.Close())
	}()

	eventStore := eventuallyfirestore.EventStore{
		Client: client,
		Serde: serde.Chain(
			user.EventProtoSerde,
			serde.NewProtoJSON(func() *userv1.Event { return new(userv1.Event) }),
		),
	}

	user.EventStoreSuite(eventStore)(t)
}

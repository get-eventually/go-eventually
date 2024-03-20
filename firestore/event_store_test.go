package firestore_test

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/require"

	eventuallyfirestore "github.com/get-eventually/go-eventually/firestore"
	"github.com/get-eventually/go-eventually/internal/user"
	"github.com/get-eventually/go-eventually/internal/user/proto"
	"github.com/get-eventually/go-eventually/serde"
)

func TestEventStore(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ctx := context.Background()

	client, err := firestore.NewClient(ctx, os.Getenv("GOOGLE_PROJECT_ID"))
	require.NoError(t, err)

	eventStore := eventuallyfirestore.EventStore{
		Client: client,
		Serde: serde.Chain(
			user.EventProtoSerde,
			serde.NewProtoJSON(func() *proto.Event { return new(proto.Event) }),
		),
	}

	user.EventStoreSuite(eventStore)(t)
}

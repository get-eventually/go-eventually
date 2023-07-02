package eventuallyfirestore_test

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/message"
	eventuallyfirestore "github.com/get-eventually/go-eventually/firestore"
	"github.com/get-eventually/go-eventually/integrationtest"
	"github.com/get-eventually/go-eventually/integrationtest/user"
	"github.com/get-eventually/go-eventually/integrationtest/user/proto"
	"github.com/get-eventually/go-eventually/serdes"
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
		Serde: serdes.Chain[message.Message, *proto.Event, []byte](
			user.EventProtoSerde,
			serdes.NewProtoJSON(func() *proto.Event { return &proto.Event{} }),
		),
	}

	integrationtest.EventStore(eventStore)(t)
}

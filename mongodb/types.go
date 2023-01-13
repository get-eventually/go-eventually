package mongodb

import (
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
)

var bsonRegistry = bsoncodec.NewRegistryBuilder().
	RegisterCodec(nil, nil).
	Build()

type persistedEvent struct {
	id      string
	version uint64
}

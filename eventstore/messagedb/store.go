package messagedb

import (
	"database/sql"
	"github.com/get-eventually/go-eventually/eventstore"
	"reflect"
)

var (
	_ eventstore.Store        = &eventstore.Appender{}
	_ eventstore.Store        = &eventstore.Streamer{}
)

type EventStore struct {
	db              *sql.DB
	eventNameToType map[string]reflect.Type
	eventTypeToName map[reflect.Type]string
}

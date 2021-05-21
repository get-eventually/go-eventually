package inmemory_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
)

func TestStoreSuite(t *testing.T) {
	suite.Run(t, eventstore.NewStoreSuite(func() eventstore.Store {
		return inmemory.NewEventStore()
	}))
}

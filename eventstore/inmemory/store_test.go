package inmemory_test

import (
	"testing"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"

	"github.com/stretchr/testify/suite"
)

func TestStoreSuite(t *testing.T) {
	suite.Run(t, eventstore.NewStoreSuite(func() eventstore.Store {
		return inmemory.NewEventStore()
	}))
}

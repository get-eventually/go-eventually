package user

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/query"
	"github.com/get-eventually/go-eventually/version"
)

// View is a public-facing representation of a User entity.
// Can be obtained through a Query handler.
type View struct {
	ID                  uuid.UUID
	Email               string
	FirstName, LastName string
	BirthDate           time.Time

	Version version.Version // NOTE: used to avoid re-processing of already-processed events.
}

// ErrNotFound is returned by a Query when a specific User has not been found.
var ErrNotFound = errors.New("user: not found")

var (
	_ query.Query                              = GetByEmail("test@email.com")
	_ query.ProcessorHandler[GetByEmail, View] = new(GetByEmailHandler)
)

// GetByEmail is a Domain Query that can be used to fetch a specific User given its email.
type GetByEmail string

// Name implements query.Query.
func (GetByEmail) Name() string { return "GetUserByEmail" }

// GetByEmailHandler is a stateful Query Handler that maintains a list of Users
// indexed by their email.
//
// It can be used to answer GetByEmail queries.
//
// GetByEmailHandler is thread-safe.
type GetByEmailHandler struct {
	mx        sync.RWMutex
	data      map[string]View
	idToEmail map[uuid.UUID]string
}

// NewGetByEmailHandler creates a new GetByEmailHandler instance.
func NewGetByEmailHandler() *GetByEmailHandler {
	handler := new(GetByEmailHandler)
	handler.data = make(map[string]View)
	handler.idToEmail = make(map[uuid.UUID]string)

	return handler
}

// Handle implements query.Handler.
func (handler *GetByEmailHandler) Handle(_ context.Context, q query.Envelope[GetByEmail]) (View, error) {
	handler.mx.RLock()
	defer handler.mx.RUnlock()

	user, ok := handler.data[string(q.Message)]
	if !ok {
		return View{}, fmt.Errorf("user.GetByEmailHandler: failed to get User by email, %w", ErrNotFound)
	}

	return user, nil
}

// Process implements event.Processor.
func (handler *GetByEmailHandler) Process(_ context.Context, evt event.Persisted) error {
	handler.mx.Lock()
	defer handler.mx.Unlock()

	userEvent, ok := evt.Message.(*Event)
	if !ok {
		return fmt.Errorf("user.GetByEmailHandler: unexpected event type, %T", evt.Message)
	}

	switch kind := userEvent.Kind.(type) {
	case *WasCreated:
		handler.idToEmail[userEvent.ID] = kind.Email
		handler.data[kind.Email] = View{
			ID:        userEvent.ID,
			Email:     kind.Email,
			FirstName: kind.FirstName,
			LastName:  kind.LastName,
			BirthDate: kind.BirthDate,
			Version:   evt.Version,
		}

	case *EmailWasUpdated:
		previousEmail, ok := handler.idToEmail[userEvent.ID]
		if !ok {
			return fmt.Errorf("user.GetByEmailHandler: expected id to have been registered, none found")
		}

		view, ok := handler.data[previousEmail]
		if !ok {
			return fmt.Errorf("user.GetByEmailHandler: expected view to be registered, none found")
		}

		if view.Version >= evt.Version {
			return nil
		}

		view.Email = kind.Email
		handler.idToEmail[userEvent.ID] = view.Email
		handler.data[view.Email] = view

	default:
		return fmt.Errorf("user.GetByEmailHandler: unexpected User event kind, %T", kind)
	}

	return nil
}

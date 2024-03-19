package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/event"
)

type CreateCommand struct {
	FirstName, LastName string
	BirthDate           time.Time
	Email               string
}

func (CreateCommand) Name() string {
	return "CreateUser"
}

type CreateCommandHandler struct {
	UUIDGenerator  func() uuid.UUID
	UserRepository aggregate.Saver[uuid.UUID, event.Event, *User]
}

func (h CreateCommandHandler) Handle(ctx context.Context, cmd command.Envelope[CreateCommand]) error {
	newUserID := h.UUIDGenerator()

	user, err := Create(newUserID, cmd.Message.FirstName, cmd.Message.LastName, cmd.Message.Email, cmd.Message.BirthDate)
	if err != nil {
		return fmt.Errorf("%T: failed to create new User, %w", h, err)
	}

	if err := h.UserRepository.Save(ctx, user); err != nil {
		return fmt.Errorf("%T: failed to save new User to repository, %w", h, err)
	}

	return nil
}

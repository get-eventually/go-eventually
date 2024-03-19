package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/command"
)

//nolint:exhaustruct // Interface implementation assertion.
var (
	_ command.Command                = CreateCommand{}
	_ command.Handler[CreateCommand] = CreateCommandHandler{}
)

// CreateCommand is a domain command that can be used to create a new User.
type CreateCommand struct {
	FirstName, LastName string
	BirthDate           time.Time
	Email               string
}

// Name implements command.Command.
func (CreateCommand) Name() string { return "CreateUser" }

// CreateCommandHandler is the command handler for CreateCommand domain commands.
type CreateCommandHandler struct {
	UUIDGenerator  func() uuid.UUID
	UserRepository aggregate.Saver[uuid.UUID, *User]
}

// Handle implements command.Handler.
func (h CreateCommandHandler) Handle(ctx context.Context, cmd command.Envelope[CreateCommand]) error {
	newUserID := h.UUIDGenerator()

	user, err := Create(newUserID, cmd.Message.FirstName, cmd.Message.LastName, cmd.Message.Email, cmd.Message.BirthDate)
	if err != nil {
		return fmt.Errorf("user.CreateCommandHandler: failed to create new User, %w", err)
	}

	if err := h.UserRepository.Save(ctx, user); err != nil {
		return fmt.Errorf("user.CreateCommandHandler: failed to save new User to repository, %w", err)
	}

	return nil
}

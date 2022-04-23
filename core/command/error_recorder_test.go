package command_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/command"
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/internal/user"
	"github.com/get-eventually/go-eventually/core/test"
)

type CreateUserFailed struct {
	Command command.Envelope[user.CreateCommand]
	Reason  string
}

func (CreateUserFailed) Name() string {
	return "CreateUserFailed"
}

func TestErrorRecorder(t *testing.T) {
	id := uuid.New()
	createUser := user.CreateCommand{
		FirstName: "John",
		LastName:  "Doe",
		BirthDate: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		Email:     "john@doe.com",
	}

	t.Run("no error recorded when the command handler doesn't fail", func(t *testing.T) {
		eventStore := test.NewInMemoryEventStore()
		userRepository := aggregate.NewEventSourcedRepository[uuid.UUID](
			eventStore,
			func() *user.User { return &user.User{} },
		)

		createUserHandler := user.CreateCommandHandler{
			UUIDGenerator:  func() uuid.UUID { return id },
			UserRepository: userRepository,
		}

		trackingEventStore := test.NewTrackingEventStore(eventStore)
		createUserHandlerWithErrorRecorder, err := command.NewErrorRecorder[user.CreateCommand](
			createUserHandler,
			command.ErrorRecorderOptions[user.CreateCommand]{
				Appender: trackingEventStore,
				EventStreamIDMapper: func(cmd command.Envelope[user.CreateCommand]) event.StreamID {
					return event.StreamID(fmt.Sprintf("user:email:%s:command-error", cmd.Message.Email))
				},
				EventMapper: func(err error, cmd command.Envelope[user.CreateCommand]) event.Envelope {
					return event.Envelope{
						Message: CreateUserFailed{
							Command: cmd,
							Reason:  err.Error(),
						},
						Metadata: nil,
					}
				},
			},
		)
		require.NoError(t, err)

		ctx := context.Background()
		assert.NoError(t, createUserHandlerWithErrorRecorder.Handle(ctx, command.Envelope[user.CreateCommand]{
			Message:  createUser,
			Metadata: nil,
		}))

		assert.Empty(t, trackingEventStore.Recorded())
	})

	t.Run("error recorded and returned when the handler fails and recorder not capturing errors", func(t *testing.T) {
		eventStore := test.NewInMemoryEventStore()
		trackingEventStore := test.NewTrackingEventStore(eventStore)

		expectedErr := fmt.Errorf("error returned for testing")
		createUserHandler := func(ctx context.Context, cmd command.Envelope[user.CreateCommand]) error {
			return expectedErr
		}

		createUserHandlerWithErrorRecorder, err := command.NewErrorRecorder[user.CreateCommand](
			command.HandlerFunc[user.CreateCommand](createUserHandler),
			command.ErrorRecorderOptions[user.CreateCommand]{
				Appender: trackingEventStore,
				EventStreamIDMapper: func(cmd command.Envelope[user.CreateCommand]) event.StreamID {
					return event.StreamID(fmt.Sprintf("user:email:%s:command-error", cmd.Message.Email))
				},
				EventMapper: func(err error, cmd command.Envelope[user.CreateCommand]) event.Envelope {
					return event.Envelope{
						Message: CreateUserFailed{
							Command: cmd,
							Reason:  err.Error(),
						},
						Metadata: nil,
					}
				},
			},
		)
		require.NoError(t, err)

		ctx := context.Background()
		cmd := command.Envelope[user.CreateCommand]{
			Message:  createUser,
			Metadata: nil,
		}

		assert.ErrorIs(t, createUserHandlerWithErrorRecorder.Handle(ctx, cmd), expectedErr)
		assert.Equal(t, trackingEventStore.Recorded(), []event.Persisted{
			{
				StreamID: event.StreamID(fmt.Sprintf("user:email:%s:command-error", createUser.Email)),
				Version:  1,
				Envelope: event.Envelope{
					Message: CreateUserFailed{
						Command: cmd,
						Reason:  expectedErr.Error(),
					},
				},
			},
		})
	})
}

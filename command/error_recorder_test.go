package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/command"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/inmemory"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

type mockCommand struct {
	message string
}

func (mockCommand) Name() string {
	return "mock_command"
}

type mockCommandHasFailed struct {
	err     error
	command mockCommand
}

func (mockCommandHasFailed) Name() string {
	return "mock_command_has_failed"
}

func TestErrorRecorder(t *testing.T) {
	t.Run("no error event is appended in case of command success", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)

		handler := command.ErrorRecorder{
			Handler: command.HandlerFunc(func(ctx context.Context, cmd eventually.Command) error {
				return nil
			}),
			Appender: trackingEventStore,
			EventMapper: func(err error, cmd eventually.Command) eventually.Payload {
				return mockCommandHasFailed{
					err:     err,
					command: cmd.Payload.(mockCommand),
				}
			},
		}

		err := handler.Handle(context.Background(), eventually.Command{
			Payload: mockCommand{message: t.Name()},
		})

		assert.NoError(t, err)
		assert.Empty(t, trackingEventStore.Recorded())
	})

	t.Run("when handler fails, record event with default stream type", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)

		expectedErr := errors.New("failed command")
		expectedCommand := eventually.Command{
			Payload: mockCommand{message: t.Name()},
		}

		handler := command.ErrorRecorder{
			Handler: command.HandlerFunc(func(ctx context.Context, cmd eventually.Command) error {
				return expectedErr
			}),
			Appender: trackingEventStore,
			EventMapper: func(err error, cmd eventually.Command) eventually.Payload {
				return mockCommandHasFailed{
					err:     err,
					command: cmd.Payload.(mockCommand),
				}
			},
		}

		err := handler.Handle(context.Background(), expectedCommand)

		assert.Error(t, err)
		assert.Equal(t, []eventstore.Event{
			{
				Version: 1,
				Stream: stream.ID{
					Type: command.FailedType,
					Name: expectedCommand.Payload.Name(),
				},
				Event: eventually.Event{
					Payload: mockCommandHasFailed{
						err:     expectedErr,
						command: expectedCommand.Payload.(mockCommand),
					},
				},
			},
		}, trackingEventStore.Recorded())
	})

	t.Run("when handler fails and CaptureError is set to true, no error is returned", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)

		expectedErr := errors.New("failed command")
		expectedCommand := eventually.Command{
			Payload: mockCommand{message: t.Name()},
		}

		handler := command.ErrorRecorder{
			CaptureErrors: true,
			Handler: command.HandlerFunc(func(ctx context.Context, cmd eventually.Command) error {
				return expectedErr
			}),
			Appender: trackingEventStore,
			EventMapper: func(err error, cmd eventually.Command) eventually.Payload {
				return mockCommandHasFailed{
					err:     err,
					command: cmd.Payload.(mockCommand),
				}
			},
		}

		err := handler.Handle(context.Background(), expectedCommand)

		assert.NoError(t, err)
		assert.Equal(t, []eventstore.Event{
			{
				Version: 1,
				Stream: stream.ID{
					Type: command.FailedType,
					Name: expectedCommand.Payload.Name(),
				},
				Event: eventually.Event{
					Payload: mockCommandHasFailed{
						err:     expectedErr,
						command: expectedCommand.Payload.(mockCommand),
					},
				},
			},
		}, trackingEventStore.Recorded())
	})

	t.Run("when handler fails, record event with custom stream type", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)

		const expectedStreamType = "mocks-command"

		expectedErr := errors.New("failed command")
		expectedCommand := eventually.Command{
			Payload: mockCommand{message: t.Name()},
		}

		handler := command.ErrorRecorder{
			Handler: command.HandlerFunc(func(ctx context.Context, cmd eventually.Command) error {
				return expectedErr
			}),
			Appender:   trackingEventStore,
			StreamType: expectedStreamType,
			EventMapper: func(err error, cmd eventually.Command) eventually.Payload {
				return mockCommandHasFailed{
					err:     err,
					command: cmd.Payload.(mockCommand),
				}
			},
		}

		err := handler.Handle(context.Background(), expectedCommand)

		assert.Error(t, err)
		assert.Equal(t, []eventstore.Event{
			{
				Version: 1,
				Stream: stream.ID{
					Type: expectedStreamType,
					Name: expectedCommand.Payload.Name(),
				},
				Event: eventually.Event{
					Payload: mockCommandHasFailed{
						err:     expectedErr,
						command: expectedCommand.Payload.(mockCommand),
					},
				},
			},
		}, trackingEventStore.Recorded())
	})

	t.Run("when handler fails, record event with custom stream name", func(t *testing.T) {
		eventStore := inmemory.NewEventStore()
		trackingEventStore := inmemory.NewTrackingEventStore(eventStore)

		expectedStreamType := "mocks-command"
		expectedErr := errors.New("failed command")
		expectedCommand := eventually.Command{
			Payload: mockCommand{message: t.Name()},
		}

		handler := command.ErrorRecorder{
			Handler: command.HandlerFunc(func(ctx context.Context, cmd eventually.Command) error {
				return expectedErr
			}),
			Appender:   trackingEventStore,
			StreamType: expectedStreamType,
			StreamNameMapper: func(cmd eventually.Command) string {
				return cmd.Payload.(mockCommand).message
			},
			EventMapper: func(err error, cmd eventually.Command) eventually.Payload {
				return mockCommandHasFailed{
					err:     err,
					command: cmd.Payload.(mockCommand),
				}
			},
		}

		err := handler.Handle(context.Background(), expectedCommand)

		assert.Error(t, err)
		assert.Equal(t, []eventstore.Event{
			{
				Version: 1,
				Stream: stream.ID{
					Type: expectedStreamType,
					Name: expectedCommand.Payload.(mockCommand).message,
				},
				Event: eventually.Event{
					Payload: mockCommandHasFailed{
						err:     expectedErr,
						command: expectedCommand.Payload.(mockCommand),
					},
				},
			},
		}, trackingEventStore.Recorded())
	})
}

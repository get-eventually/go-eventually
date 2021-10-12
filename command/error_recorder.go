package command

import (
	"context"

	"github.com/pkg/errors"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/version"
)

// FailedType is the default stream type used by ErrorRecorder
// to record failed commands to the Event Store.
const FailedType = "command-failed"

// ErrorRecorder is a command.Handler extension that records command handling
// failures on a Domain Event.
//
// This is useful for always logging command results on the Event Store,
// which would normally not be done in case of command failures,
// and to potentially build Saga Process Managers on top of these errors for
// retries and other domain-specific logic.
type ErrorRecorder struct {
	Handler

	// Appender is the Event Store instance used for appending domain events.
	Appender event.Appender

	// CaptureErrors specifies whether an error reported from the
	// command.Handler should be captured (or "silenced") by this component
	// and return nil instead.
	//
	// Default behavior is to return all errors from the command.Handler
	// to the caller.
	CaptureErrors bool

	// StreamType specifies the stream type used for the events produced by this component.
	// If unspecified, FailedCommandType will be used by default.
	StreamType string

	// StreamNameMapper maps to a StreamName value based on the command that failed.
	// This is useful in case you want to use the same Aggregate id the command is targeting.
	//
	// If unspecified, the command name will be used instead.
	StreamNameMapper func(cmd Command) string

	// EventMapper should return the Domain Event type you defined for these commands.
	//
	// NOTE: this is necessary for (de)-serialization purposes while generics are
	// still missing in Go. This requirement might be removed in the future.
	EventMapper func(err error, cmd Command) eventually.Payload
}

func (er ErrorRecorder) streamType() string {
	if er.StreamType != "" {
		return er.StreamType
	}

	return FailedType
}

func (er ErrorRecorder) buildStreamID(cmd Command) event.StreamID {
	streamName := cmd.Payload.Name()
	if er.StreamNameMapper != nil {
		streamName = er.StreamNameMapper(cmd)
	}

	return event.StreamID{
		Type: er.streamType(),
		Name: streamName,
	}
}

// Handle delegates command handling to the internal Command Handler and,
// in case of failure, appends the error and command to the Event Store
// using the user-provided EventMapper.
//
// If CaptureError has been set to "false", the error coming from the command.Handler
// will be returned to the caller.
//
// If CaptureError has been set to "true", the error from the command.Handler is silenced
// but an error can still be returned if the append operation on the Event Store fails.
func (er ErrorRecorder) Handle(ctx context.Context, cmd Command) error {
	err := er.Handler.Handle(ctx, cmd)
	if err == nil {
		return nil
	}

	streamID := er.buildStreamID(cmd)
	event := event.Event{
		Payload: er.EventMapper(err, cmd),
	}

	_, appendErr := er.Appender.Append(ctx, streamID, version.Any, event)
	if appendErr != nil && er.CaptureErrors {
		// Append error only returned if silencing command.Handler errors.
		return errors.Wrap(err, "command.ErrorRecorder: failed to append command error to event store")
	}

	if !er.CaptureErrors {
		return errors.Wrap(err, "command.ErrorRecorder: command handler failed")
	}

	return nil
}

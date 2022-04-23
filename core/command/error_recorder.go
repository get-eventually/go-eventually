package command

import (
	"context"
	"fmt"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"
)

type ErrorRecorderOptions[T Command] struct {
	// Appender is the Event Store instance used for appending domain events.
	Appender event.Appender

	// ShouldCaptureError is a function that should specify
	// whether an error reported from the command.Handler should be captured (or "silenced")
	// by this component and avoid returning it to the command caller.
	//
	// Default behavior is to return all errors from the command.Handler
	// to the caller.
	ShouldCaptureError func(err error) bool

	// EventStreamIDMapper maps to an event.StreamID value based on the command that failed.
	EventStreamIDMapper func(cmd Envelope[T]) event.StreamID

	// EventMapper should return the Domain Event type you defined for these commands.
	EventMapper func(err error, cmd Envelope[T]) event.Envelope

	// Logger is an optional logger that can be used to report append errors and such.
	Logger func(format string, args ...interface{})
}

// ErrorRecorder is a command.Handler extension that records command handling
// failures on a Domain Event.
//
// This is useful for always logging command results on the Event Store,
// which would normally not be done in case of command failures,
// and to potentially build Saga Process Managers on top of these errors for
// retries and other domain-specific logic.
type ErrorRecorder[T Command] struct {
	handler Handler[T]
	options ErrorRecorderOptions[T]
}

func NewErrorRecorder[T Command](handler Handler[T], options ErrorRecorderOptions[T]) (ErrorRecorder[T], error) {
	if handler == nil {
		return ErrorRecorder[T]{}, fmt.Errorf("command.NewErrorRecorder: handler is nil")
	}

	if options.EventStreamIDMapper == nil {
		return ErrorRecorder[T]{}, fmt.Errorf("command.NewErrorRecorder: event stream mapper func is required")
	}

	if options.EventMapper == nil {
		return ErrorRecorder[T]{}, fmt.Errorf("command.NewErrorRecorder: event mapper func is required")
	}

	return ErrorRecorder[T]{
		handler: handler,
		options: options,
	}, nil
}

// Handle delegates command handling to the internal Command Handler and,
// in case of failure, appends the error and command to the Event Store
// using the user-provided EventMapper.
//
// If ShouldCaptureError has been set and returns "false", the error coming from the command.Handler
// will be returned to the caller.
//
// If ShouldCaptureError has been set and returns "true", the error from the command.Handler is silenced
// but an error can still be returned if the append operation on the Event Store fails.
func (er ErrorRecorder[T]) Handle(ctx context.Context, cmd Envelope[T]) error {
	err := er.handler.Handle(ctx, cmd)
	if err == nil {
		return nil
	}

	shouldCaptureError := false
	if checkErr := er.options.ShouldCaptureError; checkErr != nil {
		shouldCaptureError = checkErr(err)
	}

	evt := er.options.EventMapper(err, cmd)
	eventStreamID := er.options.EventStreamIDMapper(cmd)

	if _, serr := er.options.Appender.Append(ctx, eventStreamID, version.Any, evt); serr != nil {
		// er.options.Logger.Error(er.Logger, "Failed to append command error to event store", logger.With("err", appendErr))

		if shouldCaptureError {
			// Append error only returned if silencing command.Handler errors.
			return fmt.Errorf(
				"command.ErrorRecorder: failed to append command error to event store: %w [original: %s]",
				serr, err,
			)
		}
	}

	if !shouldCaptureError {
		return fmt.Errorf("command.ErrorRecorder: command handler failed: %w", err)
	}

	return nil
}

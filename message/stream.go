package message

import (
	"errors"
	"iter"
)

// ErrAlreadyIterated is returned by Stream.Err when Iter is called more than once.
//
// The second and subsequent iterations yield nothing; Err transitions to this
// sentinel value.
var ErrAlreadyIterated = errors.New("message.Stream: already iterated")

// Stream is a single-use, iterator-backed sequence of values of type T.
//
// Producers are written as callbacks that push values via yield and return a
// terminal error if the sequence cannot be completed. Consumers iterate via
// [Stream.Iter] and check [Stream.Err] at the end:
//
//	for v := range stream.Iter() {
//		// handle v
//	}
//	if err := stream.Err(); err != nil {
//		// handle failure
//	}
//
// Consumer abandonment (break in the range loop) is NOT a failure; Err returns
// nil in that case.
//
// Stream is single-use. Calling [Stream.Iter] more than once yields an empty
// sequence and sets [Stream.Err] to [ErrAlreadyIterated].
//
// Ctx cancellation is the producer's responsibility: producers should check
// ctx.Err() at loop boundaries and return it to signal cancellation.
type Stream[T any] struct {
	produce  func(yield func(T) bool) error
	iterated bool
	err      error
}

// NewStream returns a Stream backed by the given producer.
//
// The producer is invoked lazily when [Stream.Iter] is called. It must push
// each value via yield and respect yield's bool return: if yield returns
// false, the producer should stop and return nil.
//
// A non-nil error returned by the producer is captured and surfaced via
// [Stream.Err].
func NewStream[T any](produce func(yield func(T) bool) error) *Stream[T] {
	return &Stream[T]{
		produce:  produce,
		iterated: false,
		err:      nil,
	}
}

// Iter returns an [iter.Seq] over the stream's values.
//
// Iter is single-use: calling it more than once returns an empty sequence and
// sets [Stream.Err] to [ErrAlreadyIterated].
func (s *Stream[T]) Iter() iter.Seq[T] {
	if s.iterated {
		s.err = ErrAlreadyIterated

		return func(_ func(T) bool) {}
	}

	s.iterated = true

	return func(yield func(T) bool) {
		s.err = s.produce(yield)
	}
}

// Err returns the terminal error of the stream, or nil if the stream completed
// without error (including the case of consumer abandonment).
//
// Err is valid at any time; before [Stream.Iter] is called, Err returns nil.
func (s *Stream[T]) Err() error { return s.err }

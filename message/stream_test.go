package message_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/message"
)

func TestStream_EmptyProducer(t *testing.T) {
	s := message.NewStream(func(_ func(int) bool) error {
		return nil
	})

	got := make([]int, 0)
	for v := range s.Iter() {
		got = append(got, v)
	}

	require.NoError(t, s.Err())
	assert.Empty(t, got)
}

func TestStream_YieldsAllValues(t *testing.T) {
	want := []int{1, 2, 3}

	s := message.NewStream(func(yield func(int) bool) error {
		for _, v := range want {
			if !yield(v) {
				return nil
			}
		}

		return nil
	})

	got := make([]int, 0, len(want))
	for v := range s.Iter() {
		got = append(got, v)
	}

	require.NoError(t, s.Err())
	assert.Equal(t, want, got)
}

func TestStream_ProducerError(t *testing.T) {
	wantErr := errors.New("boom")

	s := message.NewStream(func(yield func(int) bool) error {
		yield(1)

		return wantErr
	})

	got := make([]int, 0, 1)
	for v := range s.Iter() {
		got = append(got, v)
	}

	require.ErrorIs(t, s.Err(), wantErr)
	assert.Equal(t, []int{1}, got)
}

func TestStream_ConsumerAbandonment(t *testing.T) {
	s := message.NewStream(func(yield func(int) bool) error {
		for i := 1; i <= 10; i++ {
			if !yield(i) {
				return nil
			}
		}

		return nil
	})

	got := make([]int, 0, 3)
	for v := range s.Iter() {
		got = append(got, v)

		if len(got) == 3 {
			break
		}
	}

	require.NoError(t, s.Err(), "abandonment is not a failure")
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestStream_IterSingleUse(t *testing.T) {
	s := message.NewStream(func(yield func(int) bool) error {
		for _, v := range []int{1, 2, 3} {
			if !yield(v) {
				return nil
			}
		}

		return nil
	})

	// First iteration completes normally.
	got := make([]int, 0, 3)
	for v := range s.Iter() {
		got = append(got, v)
	}

	require.NoError(t, s.Err())
	assert.Equal(t, []int{1, 2, 3}, got)

	// Second iteration yields nothing and sets ErrAlreadyIterated.
	for v := range s.Iter() {
		t.Fatalf("unexpected yield on second Iter() call: %v", v)
	}

	require.ErrorIs(t, s.Err(), message.ErrAlreadyIterated)
}

func TestStream_ErrBeforeIter(t *testing.T) {
	s := message.NewStream(func(_ func(int) bool) error {
		return errors.New("unused")
	})

	require.NoError(t, s.Err(), "Err should be nil before Iter is called")
}

package event_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/version"
)

type noopMessage struct{ id int }

func (noopMessage) Name() string { return "noop" }

var _ message.Message = noopMessage{}

const testStreamID event.StreamID = "stream"

func appendN(t *testing.T, store *event.InMemoryStore, n int) {
	t.Helper()

	envelopes := make([]event.Envelope, 0, n)
	for i := range n {
		envelopes = append(envelopes, event.Envelope{
			Message:  noopMessage{id: i},
			Metadata: nil,
		})
	}

	_, err := store.Append(t.Context(), testStreamID, version.Any, envelopes...)
	require.NoError(t, err)
}

func collectIDs(stream *event.Stream) []int {
	ids := make([]int, 0)
	for evt := range stream.Iter() {
		ids = append(ids, evt.Message.(noopMessage).id) //nolint:errcheck,forcetypeassert // test helper
	}

	return ids
}

func TestInMemoryStore_Stream_EmptyStream(t *testing.T) {
	store := event.NewInMemoryStore()

	stream := store.Stream(t.Context(), "missing", version.SelectFromBeginning)

	count := 0
	for range stream.Iter() {
		count++
	}

	require.NoError(t, stream.Err())
	assert.Equal(t, 0, count)
}

func TestInMemoryStore_Stream_YieldsAllEvents(t *testing.T) {
	store := event.NewInMemoryStore()
	appendN(t, store, 3)

	stream := store.Stream(t.Context(), testStreamID, version.SelectFromBeginning)
	got := collectIDs(stream)

	require.NoError(t, stream.Err())
	assert.Equal(t, []int{0, 1, 2}, got)
}

func TestInMemoryStore_Stream_SelectorFiltersFromVersion(t *testing.T) {
	store := event.NewInMemoryStore()
	appendN(t, store, 5)

	stream := store.Stream(t.Context(), testStreamID, version.Selector{From: 3})
	got := collectIDs(stream)

	require.NoError(t, stream.Err())
	// Versions start at 1, so selector.From=3 yields events at index 2,3,4.
	assert.Equal(t, []int{2, 3, 4}, got)
}

func TestInMemoryStore_Stream_ConsumerAbandonment(t *testing.T) {
	store := event.NewInMemoryStore()
	appendN(t, store, 10)

	stream := store.Stream(t.Context(), testStreamID, version.SelectFromBeginning)

	got := make([]int, 0, 3)
	for evt := range stream.Iter() {
		//nolint:errcheck,forcetypeassert // test helper
		got = append(got, evt.Message.(noopMessage).id)

		if len(got) == 3 {
			break
		}
	}

	require.NoError(t, stream.Err(), "abandonment is not a failure")
	assert.Equal(t, []int{0, 1, 2}, got)
}

func TestInMemoryStore_Stream_ContextCancellation(t *testing.T) {
	store := event.NewInMemoryStore()
	appendN(t, store, 5)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before iteration starts

	stream := store.Stream(ctx, testStreamID, version.SelectFromBeginning)

	count := 0
	for range stream.Iter() {
		count++
	}

	require.Error(t, stream.Err())
	require.ErrorIs(t, stream.Err(), context.Canceled)
	assert.Equal(t, 0, count)
}

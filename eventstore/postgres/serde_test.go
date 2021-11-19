package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore/postgres"
	"github.com/get-eventually/go-eventually/eventstore/stream"
	"github.com/get-eventually/go-eventually/internal"
)

type event1 struct{}

//nolint:goconst // This is used for testing, it's ok not to define a constant.
func (event1) Name() string { return "test_event" }

type event2 int

func (event2) Name() string { return "test_event" }

type event3 struct {
	Field string `json:"field"`
}

func (event3) Name() string { return "test_event_3" }

func TestRegistry_Register(t *testing.T) {
	type testcase struct {
		input   []eventually.Payload
		wantErr bool
	}

	testcases := map[string]testcase{
		"no events, no registration, no failures": {},
		"registering a nil event fails": {
			input:   []eventually.Payload{nil},
			wantErr: true,
		},
		"registering a proper event works": {
			input: []eventually.Payload{internal.IntPayload(0)},
		},
		"registering the same event twice has no bad effect": {
			input: []eventually.Payload{
				internal.IntPayload(0),
				internal.IntPayload(0),
			},
		},
		"registering two different events with the same name fails": {
			input: []eventually.Payload{
				event1{},
				event2(0),
			},
			wantErr: true,
		},
	}

	for name, tc := range testcases {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			r := postgres.NewJSONRegistry()
			err := r.Register(tc.input...)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistry_Deserialize(t *testing.T) {
	type testcase struct {
		input     []byte
		eventType string
		expected  eventually.Payload
		wantErr   bool
	}

	testcases := map[string]testcase{
		"deserializing an unregistered event fails": {
			eventType: event1{}.Name(),
			wantErr:   true,
		},
		"if deserializer function fails, then failure is propagated": {
			input:     []byte("1"), // Invalid JSON representation of the event.
			eventType: event3{}.Name(),
			wantErr:   true,
		},
		"deserializer works": {
			input:     []byte(`{"field":"it works!"}`),
			eventType: event3{}.Name(),
			expected:  event3{Field: "it works!"},
		},
	}

	for name, tc := range testcases {
		name, tc := name, tc

		t.Run(name, func(t *testing.T) {
			r := postgres.NewJSONRegistry()

			require.NoError(t, r.Register(
				internal.IntPayload(0),
				internal.StringPayload(""),
				event3{},
			))

			streamID := stream.ID{
				Type: "test-type",
				Name: t.Name(),
			}

			actual, err := r.Deserialize(tc.eventType, streamID, tc.input)

			if tc.wantErr {
				assert.Nil(t, actual)
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.expected, actual)
				assert.NoError(t, err)
			}
		})
	}
}

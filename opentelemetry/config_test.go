package opentelemetry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/get-eventually/go-eventually/event"
	otelex "github.com/get-eventually/go-eventually/opentelemetry"
	"github.com/get-eventually/go-eventually/version"
)

func TestWithTracerProvider_OverridesDefault(t *testing.T) {
	h := newHarness(t)

	ies, err := otelex.NewInstrumentedEventStore(
		event.NewInMemoryStore(),
		otelex.WithTracerProvider(h.tracer),
		otelex.WithMeterProvider(h.meter),
	)
	require.NoError(t, err)

	_, err = ies.Append(t.Context(), "stream", version.Any)
	require.NoError(t, err)

	spans := h.endedSpans()
	require.Len(t, spans, 1, "expected a span to be recorded by the supplied tracer provider")
	assert.Equal(t, otelex.InstrumentationName, spans[0].InstrumentationScope().Name)
}

func TestWithMeterProvider_OverridesDefault(t *testing.T) {
	h := newHarness(t)

	ies, err := otelex.NewInstrumentedEventStore(
		event.NewInMemoryStore(),
		otelex.WithTracerProvider(h.tracer),
		otelex.WithMeterProvider(h.meter),
	)
	require.NoError(t, err)

	_, err = ies.Append(t.Context(), "stream", version.Any)
	require.NoError(t, err)

	sm := h.collectScopeMetrics(t)
	assert.Equal(t, otelex.InstrumentationName, sm.Scope.Name)
	assert.NotEmpty(t, sm.Metrics, "expected metrics on the supplied meter provider")
}

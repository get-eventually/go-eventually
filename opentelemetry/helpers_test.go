package opentelemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	otelex "github.com/get-eventually/go-eventually/opentelemetry"
	"github.com/get-eventually/go-eventually/version"
)

const exceptionEventName = "exception"

type harness struct {
	spanRecorder *tracetest.SpanRecorder
	tracer       *sdktrace.TracerProvider
	metricReader *sdkmetric.ManualReader
	meter        *sdkmetric.MeterProvider
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	t.Cleanup(func() {
		ctx := context.Background()
		require.NoError(t, tracerProvider.Shutdown(ctx))
		require.NoError(t, meterProvider.Shutdown(ctx))
	})

	return &harness{
		spanRecorder: spanRecorder,
		tracer:       tracerProvider,
		metricReader: reader,
		meter:        meterProvider,
	}
}

func (h *harness) options() []otelex.Option {
	return []otelex.Option{
		otelex.WithTracerProvider(h.tracer),
		otelex.WithMeterProvider(h.meter),
	}
}

func (h *harness) endedSpans() []sdktrace.ReadOnlySpan {
	return h.spanRecorder.Ended()
}

func (h *harness) collectScopeMetrics(t *testing.T) metricdata.ScopeMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics

	require.NoError(t, h.metricReader.Collect(context.Background(), &rm))
	require.Len(t, rm.ScopeMetrics, 1, "expected exactly one ScopeMetrics entry")

	return rm.ScopeMetrics[0]
}

func findMetric(t *testing.T, sm *metricdata.ScopeMetrics, name string) metricdata.Metrics {
	t.Helper()

	for _, m := range sm.Metrics {
		if m.Name == name {
			return m
		}
	}

	t.Fatalf("metric %q not found in scope %q", name, sm.Scope.Name)

	return metricdata.Metrics{}
}

func hasExceptionEvent(span sdktrace.ReadOnlySpan) bool {
	for _, evt := range span.Events() {
		if evt.Name == exceptionEventName {
			return true
		}
	}

	return false
}

type errorEventStore struct {
	streamErr error
	appendErr error
}

func (s *errorEventStore) Stream(_ context.Context, _ event.StreamID, _ version.Selector) *event.Stream {
	return event.NewStream(func(_ func(event.Persisted) bool) error {
		return s.streamErr
	})
}

func (s *errorEventStore) Append(
	_ context.Context,
	_ event.StreamID,
	_ version.Check,
	_ ...event.Envelope,
) (version.Version, error) {
	return 0, s.appendErr
}

type slowEventStore struct {
	inner event.Store
	delay time.Duration
}

func (s *slowEventStore) Stream(ctx context.Context, id event.StreamID, selector version.Selector) *event.Stream {
	inner := s.inner.Stream(ctx, id, selector)

	return event.NewStream(func(yield func(event.Persisted) bool) error {
		for evt := range inner.Iter() {
			time.Sleep(s.delay)

			if !yield(evt) {
				return nil
			}
		}

		return inner.Err()
	})
}

func (s *slowEventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (version.Version, error) {
	return s.inner.Append(ctx, id, expected, events...)
}

type errorRepository[I aggregate.ID, T aggregate.Root[I]] struct {
	getErr  error
	saveErr error
	result  T
}

func (r *errorRepository[I, T]) Get(_ context.Context, _ I) (T, error) {
	return r.result, r.getErr
}

func (r *errorRepository[I, T]) Save(_ context.Context, _ T) error {
	return r.saveErr
}

func newUUID(t *testing.T) uuid.UUID {
	t.Helper()

	id, err := uuid.NewRandom()
	require.NoError(t, err)

	return id
}

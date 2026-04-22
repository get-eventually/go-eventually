package opentelemetry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/message"
	"github.com/get-eventually/go-eventually/opentelemetry"
	"github.com/get-eventually/go-eventually/version"
)

type noopMessage struct{ id int }

func (noopMessage) Name() string { return "noop" }

var _ message.Message = noopMessage{}

const testStreamID event.StreamID = "test-stream"

func appendEnvelopes(t *testing.T, store event.Store, n int) {
	t.Helper()

	envelopes := make([]event.Envelope, 0, n)
	for i := range n {
		envelopes = append(envelopes, event.Envelope{Message: noopMessage{id: i}})
	}

	_, err := store.Append(t.Context(), testStreamID, version.Any, envelopes...)
	require.NoError(t, err)
}

func drainStream(t *testing.T, stream *event.Stream) {
	t.Helper()

	count := 0
	for range stream.Iter() {
		count++
	}

	require.NoError(t, stream.Err())
}

func TestNewInstrumentedEventStore_Succeeds(t *testing.T) {
	h := newHarness(t)

	ies, err := opentelemetry.NewInstrumentedEventStore(event.NewInMemoryStore(), h.options()...)
	require.NoError(t, err)
	require.NotNil(t, ies)
}

func TestStream_DelegatesAndYieldsEvents(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()
	appendEnvelopes(t, inner, 3)

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	stream := ies.Stream(t.Context(), testStreamID, version.SelectFromBeginning)

	got := make([]int, 0, 3)

	for evt := range stream.Iter() {
		msg, ok := evt.Message.(noopMessage)
		require.True(t, ok)

		got = append(got, msg.id)
	}

	require.NoError(t, stream.Err())
	assert.Equal(t, []int{0, 1, 2}, got)
}

func TestStream_RecordsSpan(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()
	appendEnvelopes(t, inner, 2)

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	drainStream(t, ies.Stream(t.Context(), testStreamID, version.Selector{From: 1}))

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, opentelemetry.EventStoreStreamSpanName, span.Name())
	assert.Equal(t, codes.Unset, span.Status().Code, "successful stream should not set error status")
	assert.Contains(t, span.Attributes(), opentelemetry.EventStreamIDKey.String(string(testStreamID)))
	assert.Contains(t, span.Attributes(), opentelemetry.EventStreamVersionSelectorKey.Int64(1))
}

func TestStream_RecordsHistogram(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()
	appendEnvelopes(t, inner, 1)

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	drainStream(t, ies.Stream(t.Context(), testStreamID, version.SelectFromBeginning))

	sm := h.collectScopeMetrics(t)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{Name: opentelemetry.InstrumentationName},
		Metrics: []metricdata.Metrics{
			{
				Name:        opentelemetry.EventStoreStreamDurationMetricName,
				Description: opentelemetry.EventStoreStreamDurationMetricDescription,
				Unit:        opentelemetry.MetricUnitMilliseconds,
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(opentelemetry.ErrorAttribute.Bool(false)),
						},
					},
				},
			},
		},
	}

	got := metricdata.ScopeMetrics{
		Scope:   sm.Scope,
		Metrics: []metricdata.Metrics{findMetric(t, &sm, opentelemetry.EventStoreStreamDurationMetricName)},
	}

	metricdatatest.AssertEqual(t, want, got,
		metricdatatest.IgnoreTimestamp(),
		metricdatatest.IgnoreValue(),
		metricdatatest.IgnoreExemplars(),
	)
}

func TestStream_DurationCoversIteration(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()
	appendEnvelopes(t, inner, 3)

	slow := &slowEventStore{inner: inner, delay: 50 * time.Millisecond}

	ies, err := opentelemetry.NewInstrumentedEventStore(slow, h.options()...)
	require.NoError(t, err)

	drainStream(t, ies.Stream(t.Context(), testStreamID, version.SelectFromBeginning))

	sm := h.collectScopeMetrics(t)
	m := findMetric(t, &sm, opentelemetry.EventStoreStreamDurationMetricName)

	data, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok, "expected Histogram[int64] data type")
	require.Len(t, data.DataPoints, 1)

	// 3 events x 50ms sleep = ~150ms. Assert the recorded sum is at least
	// 100ms to allow for scheduling jitter while still proving iteration
	// time is included.
	assert.GreaterOrEqual(t, data.DataPoints[0].Sum, int64(100),
		"recorded duration should cover full iteration, got %d ms", data.DataPoints[0].Sum)
}

func TestStream_ProducerError_RecordsErrorOnSpanAndMetric(t *testing.T) {
	h := newHarness(t)

	wantErr := errors.New("store exploded")
	inner := &errorEventStore{streamErr: wantErr}

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	stream := ies.Stream(t.Context(), testStreamID, version.SelectFromBeginning)
	for range stream.Iter() {
		t.Fatalf("no events should be yielded when the producer errors before iteration")
	}

	require.ErrorIs(t, stream.Err(), wantErr)

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, wantErr.Error(), span.Status().Description)
	assert.True(t, hasExceptionEvent(span), "span should have recorded an exception event")

	sm := h.collectScopeMetrics(t)
	m := findMetric(t, &sm, opentelemetry.EventStoreStreamDurationMetricName)

	data, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, data.DataPoints, 1)

	val, ok := data.DataPoints[0].Attributes.Value(opentelemetry.ErrorAttribute)
	require.True(t, ok, "error attribute not found on data point")
	assert.True(t, val.AsBool(), "error attribute should be true")
}

func TestStream_ConsumerAbandonment_NoErrorRecorded(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()
	appendEnvelopes(t, inner, 10)

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	stream := ies.Stream(t.Context(), testStreamID, version.SelectFromBeginning)
	count := 0

	for range stream.Iter() {
		count++
		if count == 3 {
			break
		}
	}

	require.NoError(t, stream.Err(), "abandonment is not a failure")

	spans := h.endedSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code)

	sm := h.collectScopeMetrics(t)
	m := findMetric(t, &sm, opentelemetry.EventStoreStreamDurationMetricName)

	data, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, data.DataPoints, 1)

	val, ok := data.DataPoints[0].Attributes.Value(opentelemetry.ErrorAttribute)
	require.True(t, ok)
	assert.False(t, val.AsBool(), "error attribute should be false on consumer abandonment")
}

func TestStream_SpanIsChildOfParent(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()
	appendEnvelopes(t, inner, 1)

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	ctx, parent := h.tracer.Tracer("test").Start(t.Context(), "parent")

	drainStream(t, ies.Stream(ctx, testStreamID, version.SelectFromBeginning))

	parent.End()

	spans := h.endedSpans()
	require.Len(t, spans, 2)

	// Spans are emitted in completion order: the inner Stream span finishes
	// first (inside the Iter range), then the parent.
	streamSpan, parentSpan := spans[0], spans[1]
	if streamSpan.Name() != opentelemetry.EventStoreStreamSpanName {
		streamSpan, parentSpan = spans[1], spans[0]
	}

	assert.Equal(t, opentelemetry.EventStoreStreamSpanName, streamSpan.Name())
	assert.Equal(t, "parent", parentSpan.Name())
	assert.Equal(t, parentSpan.SpanContext().SpanID(), streamSpan.Parent().SpanID())
}

func TestAppend_DelegatesAndReturnsNewVersion(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	newVersion, err := ies.Append(
		t.Context(),
		testStreamID,
		version.Any,
		event.Envelope{Message: noopMessage{id: 0}},
		event.Envelope{Message: noopMessage{id: 1}},
	)
	require.NoError(t, err)
	assert.Equal(t, version.Version(2), newVersion)

	stream := inner.Stream(t.Context(), testStreamID, version.SelectFromBeginning)
	count := 0

	for range stream.Iter() {
		count++
	}

	require.NoError(t, stream.Err())
	assert.Equal(t, 2, count)
}

func TestAppend_RecordsSpan_CheckExact(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	_, err = ies.Append(
		t.Context(),
		testStreamID,
		version.CheckExact(0),
		event.Envelope{Message: noopMessage{id: 0}},
		event.Envelope{Message: noopMessage{id: 1}},
		event.Envelope{Message: noopMessage{id: 2}},
	)
	require.NoError(t, err)

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, opentelemetry.EventStoreAppendSpanName, span.Name())
	assert.Equal(t, codes.Unset, span.Status().Code)
	assert.Contains(t, span.Attributes(), opentelemetry.EventStreamIDKey.String(string(testStreamID)))
	assert.Contains(t, span.Attributes(), opentelemetry.EventStreamExpectedVersionKey.Int64(0))
	assert.Contains(t, span.Attributes(), opentelemetry.EventStoreNumEventsKey.Int(3))
}

func TestAppend_RecordsSpan_CheckAny(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	_, err = ies.Append(t.Context(), testStreamID, version.Any)
	require.NoError(t, err)

	spans := h.endedSpans()
	require.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), opentelemetry.EventStreamExpectedVersionKey.Int64(-1))
}

func TestAppend_RecordsHistogram(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	_, err = ies.Append(t.Context(), testStreamID, version.Any, event.Envelope{Message: noopMessage{}})
	require.NoError(t, err)

	sm := h.collectScopeMetrics(t)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{Name: opentelemetry.InstrumentationName},
		Metrics: []metricdata.Metrics{
			{
				Name:        opentelemetry.EventStoreAppendDurationMetricName,
				Description: opentelemetry.EventStoreAppendDurationMetricDescription,
				Unit:        opentelemetry.MetricUnitMilliseconds,
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(opentelemetry.ErrorAttribute.Bool(false)),
						},
					},
				},
			},
		},
	}

	got := metricdata.ScopeMetrics{
		Scope:   sm.Scope,
		Metrics: []metricdata.Metrics{findMetric(t, &sm, opentelemetry.EventStoreAppendDurationMetricName)},
	}

	metricdatatest.AssertEqual(t, want, got,
		metricdatatest.IgnoreTimestamp(),
		metricdatatest.IgnoreValue(),
		metricdatatest.IgnoreExemplars(),
	)
}

func TestAppend_Error_RecordsErrorOnSpanAndMetric(t *testing.T) {
	h := newHarness(t)

	wantErr := errors.New("append failed")
	inner := &errorEventStore{appendErr: wantErr}

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	_, err = ies.Append(t.Context(), testStreamID, version.Any)
	require.ErrorIs(t, err, wantErr)

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, wantErr.Error(), span.Status().Description)
	assert.True(t, hasExceptionEvent(span), "span should have recorded an exception event")

	sm := h.collectScopeMetrics(t)
	m := findMetric(t, &sm, opentelemetry.EventStoreAppendDurationMetricName)

	data, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, data.DataPoints, 1)

	val, ok := data.DataPoints[0].Attributes.Value(opentelemetry.ErrorAttribute)
	require.True(t, ok)
	assert.True(t, val.AsBool(), "error attribute should be true on failed Append")
}

func TestAppend_SpanIsChildOfParent(t *testing.T) {
	h := newHarness(t)

	inner := event.NewInMemoryStore()

	ies, err := opentelemetry.NewInstrumentedEventStore(inner, h.options()...)
	require.NoError(t, err)

	ctx, parent := h.tracer.Tracer("test").Start(t.Context(), "parent")

	_, err = ies.Append(ctx, testStreamID, version.Any)
	require.NoError(t, err)

	parent.End()

	spans := h.endedSpans()
	require.Len(t, spans, 2)

	appendSpan, parentSpan := spans[0], spans[1]
	if appendSpan.Name() != opentelemetry.EventStoreAppendSpanName {
		appendSpan, parentSpan = spans[1], spans[0]
	}

	assert.Equal(t, opentelemetry.EventStoreAppendSpanName, appendSpan.Name())
	assert.Equal(t, "parent", parentSpan.Name())
	assert.Equal(t, parentSpan.SpanContext().SpanID(), appendSpan.Parent().SpanID())
}

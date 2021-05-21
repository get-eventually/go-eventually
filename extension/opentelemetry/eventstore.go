package opentelemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
)

var _ EventStore = EventStoreWrapper{}

// EventStore is the Event Store interface this package decorates.
type EventStore interface {
	eventstore.Appender
	eventstore.Streamer
}

// EventStoreWrapper is a wrapper to provide OpenTelemetry instrumentation
// for EventStore compatible implementations, and compatible
// with the same interface to be used seamlessly in your pre-existing code.
//
// Use WrapEventStore to create new instance of this type.
type EventStoreWrapper struct {
	eventStore   EventStore
	tracer       trace.Tracer
	appendMetric metric.Int64UpDownCounter
}

// WrapEventStore creates a new EventStoreWrapper instance, using the provided
// TracerProvider and MeterProvider to collect traces and metrics.
func WrapEventStore(
	es EventStore,
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
) (EventStoreWrapper, error) {
	meter := meterProvider.Meter(instrumentationName)

	appendMetric, err := meter.NewInt64UpDownCounter(AppendMetric)
	if err != nil {
		return EventStoreWrapper{}, fmt.Errorf("opentelemetry.WrapEventStore: failed to create append metric: %w", err)
	}

	return EventStoreWrapper{
		eventStore:   es,
		tracer:       tracerProvider.Tracer(instrumentationName),
		appendMetric: appendMetric,
	}, nil
}

// StreamAll delegates the call to the underlying Event Store and records
// a trace of the result.
func (sw EventStoreWrapper) StreamAll(ctx context.Context, es eventstore.EventStream, selectt eventstore.Select) error {
	ctx, span := sw.tracer.Start(ctx, StreamAllSpanName, trace.WithAttributes(
		SelectFromAttribute.Int64(selectt.From),
	))
	defer span.End()

	err := sw.eventStore.StreamAll(ctx, es, selectt)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

// StreamByType delegates the call to the underlying Event Store and records
// a trace of the result.
func (sw EventStoreWrapper) StreamByType(
	ctx context.Context,
	es eventstore.EventStream,
	typ string,
	selectt eventstore.Select,
) error {
	ctx, span := sw.tracer.Start(ctx, StreamByTypeSpanName, trace.WithAttributes(
		StreamTypeAttribute.String(typ),
		SelectFromAttribute.Int64(selectt.From),
	))
	defer span.End()

	err := sw.eventStore.StreamByType(ctx, es, typ, selectt)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

// Stream delegates the call to the underlying Event Store and records
// a trace of the result.
func (sw EventStoreWrapper) Stream(
	ctx context.Context,
	es eventstore.EventStream,
	id eventstore.StreamID,
	selectt eventstore.Select,
) error {
	ctx, span := sw.tracer.Start(ctx, StreamSpanName, trace.WithAttributes(
		StreamTypeAttribute.String(id.Type),
		StreamNameAttribute.String(id.Name),
		SelectFromAttribute.Int64(selectt.From),
	))
	defer span.End()

	err := sw.eventStore.Stream(ctx, es, id, selectt)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

// Append delegates the call to the underlying Event Store and records
// a trace of the result and increments the metric referenced by AppendMetric.
func (sw EventStoreWrapper) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	ctx, span := sw.tracer.Start(ctx, AppendSpanName, trace.WithAttributes(
		StreamTypeAttribute.String(id.Type),
		StreamNameAttribute.String(id.Name),
		VersionCheckAttribute.Int64(int64(expected)),
	))
	defer span.End()

	newVersion, err := sw.eventStore.Append(ctx, id, expected, events...)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(VersionNewAttribute.Int64(newVersion))
	}

	sw.reportAppendMetrics(ctx, events...)

	return newVersion, err
}

func (sw EventStoreWrapper) reportAppendMetrics(ctx context.Context, events ...eventually.Event) {
	for _, event := range events {
		sw.appendMetric.
			Add(ctx, 1, EventTypeAttribute.String(event.Payload.Name()))
	}
}

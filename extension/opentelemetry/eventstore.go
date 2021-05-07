package opentelemetry

import (
	"context"
	"fmt"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var _ eventstore.Store = EventStoreWrapper{}

// EventStoreWrapper is a wrapper to provide OpenTelemetry instrumentation
// for eventstore.Store compatible implementations, and compatible
// with the same interface to be used seamlessly in your pre-existing code.
//
// Use WrapEventStore to create new instance of this type.
type EventStoreWrapper struct {
	eventstore.Store

	tracer       trace.Tracer
	appendMetric metric.Int64UpDownCounter
}

// WrapEventStore creates a new EventStoreWrapper instance, using the provided
// TracerProvider and MeterProvider to collect traces and metrics.
func WrapEventStore(
	es eventstore.Store,
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
) (EventStoreWrapper, error) {
	meter := meterProvider.Meter(instrumentationName)

	appendMetric, err := meter.NewInt64UpDownCounter(AppendMetric)
	if err != nil {
		return EventStoreWrapper{}, fmt.Errorf("opentelemetry.WrapEventStore: failed to create append metric: %w", err)
	}

	return EventStoreWrapper{
		Store:        es,
		tracer:       tracerProvider.Tracer(instrumentationName),
		appendMetric: appendMetric,
	}, nil
}

// Stream delegates the call to the underlying Event Store and records
// a trace of the result.
func (sw EventStoreWrapper) StreamAll(ctx context.Context, es eventstore.EventStream, selectt eventstore.Select) error {
	ctx, span := sw.tracer.Start(ctx, StreamAllSpanName, trace.WithAttributes(
		SelectFromLabel.Int64(selectt.From),
	))
	defer span.End()

	err := sw.Store.StreamAll(ctx, es, selectt)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (sw EventStoreWrapper) StreamByType(
	ctx context.Context,
	es eventstore.EventStream,
	typ string,
	selectt eventstore.Select,
) error {
	ctx, span := sw.tracer.Start(ctx, StreamByTypeSpanName, trace.WithAttributes(
		StreamTypeLabel.String(typ),
		SelectFromLabel.Int64(selectt.From),
	))
	defer span.End()

	err := sw.Store.StreamByType(ctx, es, typ, selectt)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (sw EventStoreWrapper) Stream(
	ctx context.Context,
	es eventstore.EventStream,
	id eventstore.StreamID,
	selectt eventstore.Select,
) error {
	ctx, span := sw.tracer.Start(ctx, StreamSpanName, trace.WithAttributes(
		StreamTypeLabel.String(id.Type),
		StreamNameLabel.String(id.Name),
		SelectFromLabel.Int64(selectt.From),
	))
	defer span.End()

	err := sw.Store.Stream(ctx, es, id, selectt)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (sw EventStoreWrapper) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	ctx, span := sw.tracer.Start(ctx, AppendSpanName, trace.WithAttributes(
		StreamTypeLabel.String(id.Type),
		StreamNameLabel.String(id.Name),
		VersionCheckLabel.Int64(int64(expected)),
	))
	defer span.End()

	newVersion, err := sw.Store.Append(ctx, id, expected, events...)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(VersionNewLabel.Int64(newVersion))
	}

	sw.reportAppendMetrics(ctx, events...)

	return newVersion, err
}

func (sw EventStoreWrapper) reportAppendMetrics(ctx context.Context, events ...eventually.Event) {
	for _, event := range events {
		sw.appendMetric.
			Add(ctx, 1, EventTypeLabel.String(event.Payload.Name()))
	}
}

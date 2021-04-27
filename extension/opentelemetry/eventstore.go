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

type eventStoreTelemetry struct {
	tracer       trace.Tracer
	appendMetric metric.Int64UpDownCounter
}

// EventStoreWrapper is a wrapper to provide OpenTelemetry instrumentation
// for eventstore.Store compatible implementations, and compatible
// with the same interface to be used seamlessly in your pre-existing code.
//
// Use WrapEventStore to create new instance of this type.
type EventStoreWrapper struct {
	eventstore.Store
	eventStoreTelemetry
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
		Store: es,
		eventStoreTelemetry: eventStoreTelemetry{
			tracer:       tracerProvider.Tracer(instrumentationName),
			appendMetric: appendMetric,
		},
	}, nil
}

// Stream delegates the call to the underlying Event Store and records
// a trace of the result.
func (sw EventStoreWrapper) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	ctx, span := sw.tracer.Start(ctx, StreamSpanName, trace.WithAttributes(
		StreamTypeLabel.String("*"),
		StreamFromLabel.Int64(from),
	))
	defer span.End()

	err := sw.Store.Stream(ctx, es, from)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

// Type returns a evenstore.Typed access with tracing instrumentation,
// using the underlying Event Store.
func (sw EventStoreWrapper) Type(ctx context.Context, typ string) (eventstore.Typed, error) {
	tsw, err := sw.Store.Type(ctx, typ)
	if err != nil {
		return nil, err
	}

	return typedEventStoreWrapper{
		Typed:               tsw,
		eventStoreTelemetry: sw.eventStoreTelemetry,
		streamType:          typ,
	}, nil
}

type typedEventStoreWrapper struct {
	eventstore.Typed
	eventStoreTelemetry

	streamType string
}

func (tsw typedEventStoreWrapper) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	ctx, span := tsw.tracer.Start(ctx, StreamSpanName, trace.WithAttributes(
		StreamTypeLabel.String(tsw.streamType),
		StreamFromLabel.Int64(from),
	))
	defer span.End()

	err := tsw.Typed.Stream(ctx, es, from)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (tsw typedEventStoreWrapper) Instance(id string) eventstore.Instanced {
	return &instancedEventStoreWrapper{
		Instanced:           tsw.Typed.Instance(id),
		eventStoreTelemetry: tsw.eventStoreTelemetry,
		streamType:          tsw.streamType,
		streamName:          id,
	}
}

// NOTE(ar3s3ru): this type uses pointer semantics to save up some memory,
// since the gocritic linter complaints :D
type instancedEventStoreWrapper struct {
	eventstore.Instanced
	eventStoreTelemetry

	streamType string
	streamName string
}

func (isw *instancedEventStoreWrapper) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	ctx, span := isw.tracer.Start(ctx, StreamSpanName, trace.WithAttributes(
		StreamTypeLabel.String(isw.streamType),
		StreamNameLabel.String(isw.streamName),
		StreamFromLabel.Int64(from),
	))
	defer span.End()

	err := isw.Instanced.Stream(ctx, es, from)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (isw *instancedEventStoreWrapper) Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error) {
	ctx, span := isw.tracer.Start(ctx, AppendSpanName, trace.WithAttributes(
		StreamTypeLabel.String(isw.streamType),
		StreamNameLabel.String(isw.streamName),
		VersionCheckLabel.Int64(version),
	))
	defer span.End()

	newVersion, err := isw.Instanced.Append(ctx, version, events...)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(VersionNewLabel.Int64(newVersion))
	}

	isw.reportAppendMetrics(ctx, events...)

	return newVersion, err
}

func (isw *instancedEventStoreWrapper) reportAppendMetrics(ctx context.Context, events ...eventually.Event) {
	for _, event := range events {
		isw.appendMetric.
			Add(ctx, 1, EventTypeLabel.String(event.Payload.Name()))
	}
}

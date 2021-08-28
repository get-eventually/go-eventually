package oteleventually

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
)

var _ eventstore.Store = &InstrumentedEventStore{}

// InstrumentedEventStore is a wrapper to provide OpenTelemetry instrumentation
// for EventStore compatible implementations, and compatible
// with the same interface to be used seamlessly in your pre-existing code.
//
// Use InstrumentEventStore to create new instance of this type.
type InstrumentedEventStore struct {
	meter      metric.Meter
	tracer     trace.Tracer
	eventStore eventstore.Store

	appendCount    metric.Int64Counter
	appendDuration metric.Int64ValueRecorder

	streamCount    metric.Int64Counter
	streamDuration metric.Int64ValueRecorder
}

func (es *InstrumentedEventStore) registerMetrics() error {
	var err error

	if es.appendCount, err = es.meter.NewInt64Counter(
		"eventually.events.append.count",
		metric.WithDescription("Count of append operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	if es.appendDuration, err = es.meter.NewInt64ValueRecorder(
		"eventually.events.append.duration.milliseconds",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("Duration in milliseconds of append operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	if es.streamCount, err = es.meter.NewInt64Counter(
		"eventually.events.stream.count",
		metric.WithDescription("Count of stream operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	if es.streamDuration, err = es.meter.NewInt64ValueRecorder(
		"eventually.events.stream.duration.milliseconds",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("Duration in milliseconds of stream operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	return nil
}

// InstrumentEventStore creates a new InstrumentedEventStore instance.
func InstrumentEventStore(es eventstore.Store, opts ...Option) (*InstrumentedEventStore, error) {
	cfg := newConfig(opts...)

	ies := &InstrumentedEventStore{
		meter:      cfg.meter(),
		tracer:     cfg.tracer(),
		eventStore: es,
	}

	if err := ies.registerMetrics(); err != nil {
		return nil, err
	}

	return ies, nil
}

func (es *InstrumentedEventStore) reportStreamMetrics(
	ctx context.Context,
	start time.Time,
	err error,
	attributes ...attribute.KeyValue,
) {
	es.streamCount.Add(ctx, 1, append(attributes, ErrorAttribute.Bool(err != nil))...)
	es.streamDuration.Record(ctx, time.Since(start).Milliseconds(), attributes...)
}

// StreamAll delegates the call to the underlying Event Store and records
// a trace of the result.
func (es *InstrumentedEventStore) StreamAll(
	ctx context.Context,
	stream eventstore.EventStream,
	selectt eventstore.Select,
) (err error) {
	attributes := []attribute.KeyValue{
		SelectFromAttribute.Int64(selectt.From),
	}

	start := time.Now()
	defer func() {
		es.reportStreamMetrics(ctx, start, err, append(attributes,
			StreamNameAttribute.String("*"),
			StreamTypeAttribute.String("*"),
		)...)
	}()

	ctx, span := es.tracer.Start(ctx, StreamAllSpanName, trace.WithAttributes(attributes...))
	defer span.End()

	if err = es.eventStore.StreamAll(ctx, stream, selectt); err != nil {
		span.RecordError(err)
	}

	return err
}

// StreamByType delegates the call to the underlying Event Store and records
// a trace of the result.
func (es *InstrumentedEventStore) StreamByType(
	ctx context.Context,
	stream eventstore.EventStream,
	typ string,
	selectt eventstore.Select,
) (err error) {
	attributes := []attribute.KeyValue{
		SelectFromAttribute.Int64(selectt.From),
		StreamTypeAttribute.String(typ),
	}

	ctx, span := es.tracer.Start(ctx, StreamByTypeSpanName, trace.WithAttributes(attributes...))
	defer span.End()

	start := time.Now()
	defer func() {
		es.reportStreamMetrics(ctx, start, err, append(attributes,
			StreamNameAttribute.String("*"),
		)...)
	}()

	if err = es.eventStore.StreamByType(ctx, stream, typ, selectt); err != nil {
		span.RecordError(err)
	}

	return err
}

// Stream delegates the call to the underlying Event Store and records
// a trace of the result.
func (es *InstrumentedEventStore) Stream(
	ctx context.Context,
	stream eventstore.EventStream,
	id eventstore.StreamID,
	selectt eventstore.Select,
) (err error) {
	attributes := []attribute.KeyValue{
		SelectFromAttribute.Int64(selectt.From),
		StreamTypeAttribute.String(id.Type),
		StreamNameAttribute.String(id.Name),
	}

	start := time.Now()
	defer func() {
		es.reportStreamMetrics(ctx, start, err, attributes...)
	}()

	ctx, span := es.tracer.Start(ctx, StreamSpanName, trace.WithAttributes(attributes...))
	defer span.End()

	if err = es.eventStore.Stream(ctx, stream, id, selectt); err != nil {
		span.RecordError(err)
	}

	return err
}

// Append delegates the call to the underlying Event Store and records
// a trace of the result and increments the metric referenced by AppendMetric.
func (es *InstrumentedEventStore) Append(
	ctx context.Context,
	id eventstore.StreamID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (newVersion int64, err error) {
	attributes := []attribute.KeyValue{
		StreamTypeAttribute.String(id.Type),
		StreamNameAttribute.String(id.Name),
		VersionCheckAttribute.Int64(int64(expected)),
		attribute.Any("events", events),
	}

	start := time.Now()
	defer func() {
		es.appendDuration.Record(ctx, time.Since(start).Milliseconds())

		attributes := []attribute.KeyValue{ErrorAttribute.Bool(err != nil)} //nolint:govet // Shadowing here is fine.
		for _, event := range events {
			es.appendCount.Add(ctx, 1, append(attributes, EventTypeAttribute.String(event.Payload.Name()))...)
		}
	}()

	ctx, span := es.tracer.Start(ctx, AppendSpanName, trace.WithAttributes(attributes...))
	defer span.End()

	if newVersion, err = es.eventStore.Append(ctx, id, expected, events...); err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(VersionNewAttribute.Int64(newVersion))
	}

	return newVersion, err
}

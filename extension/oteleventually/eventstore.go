package oteleventually

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/eventstore/stream"
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
	appendDuration metric.Int64Histogram

	streamCount    metric.Int64Counter
	streamDuration metric.Int64Histogram
}

func (es *InstrumentedEventStore) registerMetrics() error {
	var err error

	if es.appendCount, err = es.meter.NewInt64Counter(
		"eventually.events.append.count",
		metric.WithDescription("Count of append operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	if es.appendDuration, err = es.meter.NewInt64Histogram(
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

	if es.streamDuration, err = es.meter.NewInt64Histogram(
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

// Stream delegates the call to the underlying Event Store and records
// a trace of the result.
func (es *InstrumentedEventStore) Stream(
	ctx context.Context,
	eventStream eventstore.EventStream,
	target stream.Target,
	selectt eventstore.Select,
) (err error) {
	attributes := []attribute.KeyValue{
		SelectFromAttribute.Int64(selectt.From),
	}

	switch t := target.(type) {
	case stream.All:
		attributes = append(attributes, StreamTargetAttribute.String("<all>"))
	case stream.ByType:
		attributes = append(attributes, StreamTypeAttribute.String(string(t)))
	case stream.ByTypes:
		attributes = append(attributes, StreamTypeAttribute.StringSlice(t))
	case stream.ByID:
		attributes = append(attributes, StreamTypeAttribute.String(t.Type), StreamNameAttribute.String(t.Name))
	}

	start := time.Now()
	defer func() {
		es.reportStreamMetrics(ctx, start, err, attributes...)
	}()

	ctx, span := es.tracer.Start(ctx, "EventStore.Stream", trace.WithAttributes(attributes...))
	defer span.End()

	if err = es.eventStore.Stream(ctx, eventStream, target, selectt); err != nil {
		span.RecordError(err)
	}

	return err
}

// Append delegates the call to the underlying Event Store and records
// a trace of the result and increments the metric referenced by AppendMetric.
func (es *InstrumentedEventStore) Append(
	ctx context.Context,
	id stream.ID,
	expected eventstore.VersionCheck,
	events ...eventually.Event,
) (newVersion int64, err error) {
	attributes := []attribute.KeyValue{
		StreamTypeAttribute.String(id.Type),
		StreamNameAttribute.String(id.Name),
		VersionCheckAttribute.Int64(int64(expected)),
	}

	start := time.Now()
	defer func() {
		es.appendDuration.Record(ctx, time.Since(start).Milliseconds())

		attributes := []attribute.KeyValue{ErrorAttribute.Bool(err != nil)} //nolint:govet // Shadowing here is fine.
		for _, event := range events {
			es.appendCount.Add(ctx, 1, append(attributes, EventTypeAttribute.String(event.Payload.Name()))...)
		}
	}()

	spanAttributes := attributes
	if b, err := json.Marshal(events); b != nil && err == nil { //nolint:govet // err shadowing is fine.
		spanAttributes = append(spanAttributes, attribute.String("events", string(b)))
	}

	ctx, span := es.tracer.Start(ctx, "EventStore.Append", trace.WithAttributes(spanAttributes...))
	defer span.End()

	if newVersion, err = es.eventStore.Append(ctx, id, expected, events...); err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(VersionNewAttribute.Int64(newVersion))
	}

	return newVersion, err
}

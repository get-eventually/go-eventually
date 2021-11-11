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

	"github.com/get-eventually/go-eventually/event"
)

var _ event.Processor = &InstrumentedProcessor{}

// InstrumentedProcessor wraps an event.Processor instance to provide
// telemetry support using OpenTelemetry.
//
// Use InstrumentProjection function to create a new instance.
type InstrumentedProcessor struct {
	name      string
	meter     metric.Meter
	tracer    trace.Tracer
	processor event.Processor

	count    metric.Int64Counter
	duration metric.Int64Histogram
}

func (ip *InstrumentedProcessor) registerMetrics() error {
	var err error

	if ip.count, err = ip.meter.NewInt64Counter(
		"eventually.projection.count",
		metric.WithDescription("Count of projection operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	if ip.duration, err = ip.meter.NewInt64Histogram(
		"eventually.projection.duration.milliseconds",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("Duration in milliseconds of projection operations performed"),
	); err != nil {
		return fmt.Errorf("oteleventually: failed to register metric: %w", err)
	}

	return nil
}

// InstrumentProjection wraps a Projection Applier to provide support for exporting
// telemetry data using OpenTelemetry.
//
// The name provided will be used for both traces and metrics exported.
func InstrumentProjection(processor event.Processor, opts ...Option) (*InstrumentedProcessor, error) {
	cfg := newConfig(opts...)

	ip := &InstrumentedProcessor{
		meter:     cfg.meter(),
		tracer:    cfg.tracer(),
		name:      cfg.ProjectionName,
		processor: processor,
	}

	if err := ip.registerMetrics(); err != nil {
		return nil, err
	}

	return ip, nil
}

// Process processes the provided Event using the underlying event.Processor
// implementation, and reports telemetry data on its execution.
func (ip *InstrumentedProcessor) Process(ctx context.Context, evt event.Persisted) (err error) {
	attributes := []attribute.KeyValue{
		ProcessorNameAttribute.String(ip.name),
		EventTypeAttribute.String(evt.Payload.Name()),
	}

	spanAttributes := append(attributes, //nolint:gocritic // Intended behavior.
		StreamTypeAttribute.String(evt.Stream.Type),
		StreamNameAttribute.String(evt.Stream.Name),
		EventVersionAttribute.Int64(int64(evt.Version)),
		EventSequenceNumberAttribute.Int64(int64(evt.SequenceNumber)),
	)

	if eventStr, err := json.Marshal(evt.Event); err == nil { //nolint:govet // Shadowing of the error is fine.
		spanAttributes = append(spanAttributes, attribute.String("event", string(eventStr)))
	}

	ctx, span := ip.tracer.Start(ctx, "Projection.Apply", trace.WithAttributes(spanAttributes...))
	defer span.End()

	start := time.Now()
	defer func() {
		ip.duration.Record(ctx, time.Since(start).Milliseconds(), attributes...)
		ip.count.Add(ctx, 1, append(attributes, ErrorAttribute.Bool(err != nil))...)
	}()

	if err = ip.processor.Process(ctx, evt); err != nil {
		span.RecordError(err)
	}

	return err
}

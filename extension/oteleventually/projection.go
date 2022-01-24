package oteleventually

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/projection"
)

var _ projection.Applier = &InstrumentedProjection{}

// InstrumentedProjection wraps a projection.Applier instance to provide
// telemetry support using OpenTelemetry.
//
// Use InstrumentProjection function to create a new instance.
type InstrumentedProjection struct {
	name    string
	meter   metric.Meter
	tracer  trace.Tracer
	applier projection.Applier

	count    metric.Int64Counter
	duration metric.Int64Histogram
}

func (ip *InstrumentedProjection) registerMetrics() error {
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
func InstrumentProjection(applier projection.Applier, opts ...Option) (*InstrumentedProjection, error) {
	cfg := newConfig(opts...)

	ip := &InstrumentedProjection{
		meter:   cfg.meter(),
		tracer:  cfg.tracer(),
		name:    cfg.ProjectionName,
		applier: applier,
	}

	if err := ip.registerMetrics(); err != nil {
		return nil, err
	}

	return ip, nil
}

// Apply processes the provided Event using the underlying projection.Applier
// implementation, and reports telemetry data on its execution.
func (ip *InstrumentedProjection) Apply(ctx context.Context, event eventstore.Event) (err error) {
	attributes := []attribute.KeyValue{
		ProjectionNameAttribute.String(ip.name),
		EventTypeAttribute.String(event.Payload.Name()),
	}

	spanAttributes := append(attributes, //nolint:gocritic // Intended behavior.
		StreamTypeAttribute.String(event.Stream.Type),
		StreamNameAttribute.String(event.Stream.Name),
		EventVersionAttribute.Int64(event.Version),
	)

	ctx, span := ip.tracer.Start(ctx, "Projection.Apply", trace.WithAttributes(spanAttributes...))
	defer span.End()

	start := time.Now()
	defer func() {
		ip.duration.Record(ctx, time.Since(start).Milliseconds(), attributes...)
		ip.count.Add(ctx, 1, append(attributes, ErrorAttribute.Bool(err != nil))...)
	}()

	if err = ip.applier.Apply(ctx, event); err != nil {
		span.RecordError(err)
	}

	return err
}

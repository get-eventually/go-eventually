package opentelemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually/eventstore"
	"github.com/get-eventually/go-eventually/projection"
)

var _ projection.Applier = ProjectionWrapper{}

// ProjectionWrapper wraps a projection.Applier instance to provide
// telemetry support using OpenTelemetry.
//
// Use WrapProjection function to create a new instance.
type ProjectionWrapper struct {
	name           string
	applier        projection.Applier
	tracer         trace.Tracer
	durationMetric metric.Int64ValueRecorder
}

// WrapProjection wraps a Projection Applier to provide support for exporting
// telemetry data using OpenTelemetry.
//
// The name provided will be used for both traces and metrics exported.
func WrapProjection(
	name string,
	applier projection.Applier,
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
) (ProjectionWrapper, error) {
	meter := meterProvider.Meter(instrumentationName)

	durationMetric, err := meter.NewInt64ValueRecorder(ProjectionApplyMetric)
	if err != nil {
		return ProjectionWrapper{}, fmt.Errorf("opentelemetry.WrapProjection: failed to create metric: %w", err)
	}

	return ProjectionWrapper{
		name:           name,
		applier:        applier,
		tracer:         tracerProvider.Tracer(instrumentationName),
		durationMetric: durationMetric,
	}, nil
}

// Apply processes the provided Event using the underlying projection.Applier
// implementation, and reports telemetry data on its execution.
func (pw ProjectionWrapper) Apply(ctx context.Context, event eventstore.Event) error {
	ctx, span := pw.tracer.Start(ctx, ApplierSpanName, trace.WithAttributes(
		ProjectionNameAttribute.String(pw.name),
		StreamTypeAttribute.String(event.Stream.Type),
		StreamNameAttribute.String(event.Stream.Name),
		EventTypeAttribute.String(event.Payload.Name()),
		EventVersionAttribute.Int64(event.Version),
	))
	defer span.End()

	start := time.Now()

	err := pw.applier.Apply(ctx, event)
	if err != nil {
		span.RecordError(err)
	}

	pw.durationMetric.Record(ctx, time.Since(start).Milliseconds(),
		ProjectionNameAttribute.String(pw.name),
	)

	return err
}

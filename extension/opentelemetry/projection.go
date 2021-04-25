package opentelemetry

import (
	"context"

	"github.com/eventually-rs/eventually-go/eventstore"
	"github.com/eventually-rs/eventually-go/projection"

	"go.opentelemetry.io/otel/trace"
)

var _ projection.Applier = ProjectionWrapper{}

type ProjectionWrapper struct {
	name    string
	applier projection.Applier
	tracer  trace.Tracer
}

func WrapProjection(name string, applier projection.Applier, tracerProvider trace.TracerProvider) ProjectionWrapper {
	return ProjectionWrapper{
		name:    name,
		applier: applier,
		tracer:  tracerProvider.Tracer(instrumentationName),
	}
}

func (pw ProjectionWrapper) Apply(ctx context.Context, event eventstore.Event) error {
	ctx, span := pw.tracer.Start(ctx, ApplierSpanName, trace.WithAttributes(
		ProjectionNameLabel.String(pw.name),
		StreamTypeLabel.String(event.StreamType),
		StreamNameLabel.String(event.StreamName),
		EventTypeLabel.String(event.Payload.Name()),
		EventVersionLabel.Int64(event.Version),
	))
	defer span.End()

	err := pw.applier.Apply(ctx, event)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

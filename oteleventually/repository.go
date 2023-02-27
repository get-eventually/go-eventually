package oteleventually

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually/core/aggregate"
)

// Attribute keys used by the InstrumentedRepository instrumentation.
const (
	ErrorAttribute            attribute.Key = "error"
	AggregateTypeAttribute    attribute.Key = "aggregate.type"
	AggregateVersionAttribute attribute.Key = "aggregate.version"
	AggregateIDAttribute      attribute.Key = "aggregate.id"
)

// InstrumentedRepository is a wrapper type over an aggregate.Repository
// instance to provide instrumentation, in the form of metrics and traces
// using OpenTelemetry.
//
// Use NewInstrumentedRepository for constructing a new instance of this type.
type InstrumentedRepository[I aggregate.ID, T aggregate.Root[I]] struct {
	aggregateType aggregate.Type[I, T]
	repository    aggregate.Repository[I, T]

	tracer       trace.Tracer
	getDuration  instrument.Int64Histogram
	saveDuration instrument.Int64Histogram
}

func (ir *InstrumentedRepository[I, T]) registerMetrics(meter metric.Meter) error {
	var err error

	if ir.getDuration, err = meter.Int64Histogram(
		"eventually.repository.get.duration.milliseconds",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("Duration in milliseconds of aggregate.Repository.Get operations performed."),
	); err != nil {
		return fmt.Errorf("oteleventually.InstrumentedRepository: failed to register metric: %w", err)
	}

	if ir.saveDuration, err = meter.Int64Histogram(
		"eventually.repository.save.duration.milliseconds",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("Duration in milliseconds of aggregate.Repository.Save operations performed."),
	); err != nil {
		return fmt.Errorf("oteleventually.InstrumentedRepository: failed to register metric: %w", err)
	}

	return nil
}

// NewInstrumentedRepository returns a wrapper type to provide OpenTelemetry
// instrumentation (metrics and traces) around an aggregate.Repository.
//
// The aggregate.Type for the Repository is also used for reporting the
// Aggregate Type name as an attribute.
//
// An error is returned if metrics could not be registered.
func NewInstrumentedRepository[I aggregate.ID, T aggregate.Root[I]](
	aggregateType aggregate.Type[I, T],
	repository aggregate.Repository[I, T],
	options ...Option,
) (*InstrumentedRepository[I, T], error) {
	cfg := newConfig(options...)

	ir := &InstrumentedRepository[I, T]{
		aggregateType: aggregateType,
		repository:    repository,
		tracer:        cfg.tracer(),
	}

	if err := ir.registerMetrics(cfg.meter()); err != nil {
		return nil, err
	}

	return ir, nil
}

// Get calls the wrapped aggregate.Repository.Get method and records metrics
// and traces around it.
func (ir *InstrumentedRepository[I, T]) Get(ctx context.Context, id I) (result T, err error) {
	attributes := []attribute.KeyValue{
		AggregateTypeAttribute.String(ir.aggregateType.Name),
	}

	//nolint:gocritic // Not appending to the same slice done on purpose.
	spanAttributes := append(attributes,
		AggregateIDAttribute.String(id.String()),
	)

	ctx, span := ir.tracer.Start(ctx, "aggregate.Repository.Get", trace.WithAttributes(spanAttributes...))
	start := time.Now()

	defer func() {
		attributes := append(attributes, ErrorAttribute.Bool(err != nil))

		duration := time.Since(start)
		ir.getDuration.Record(ctx, duration.Milliseconds(), attributes...)

		if err != nil {
			span.RecordError(err)
		}

		span.End()
	}()

	result, err = ir.repository.Get(ctx, id)

	return
}

// Save calls the wrapped aggregate.Repository.Save method and records metrics
// and traces around it.
func (ir *InstrumentedRepository[I, T]) Save(ctx context.Context, root T) (err error) {
	attributes := []attribute.KeyValue{
		AggregateTypeAttribute.String(ir.aggregateType.Name),
	}

	//nolint:gocritic // Not appending to the same slice done on purpose.
	spanAttributes := append(attributes,
		AggregateIDAttribute.String(root.AggregateID().String()),
		AggregateVersionAttribute.Int64(int64(root.Version())),
	)

	ctx, span := ir.tracer.Start(ctx, "aggregate.Repository.Save", trace.WithAttributes(spanAttributes...))
	start := time.Now()

	defer func() {
		attributes := append(attributes, ErrorAttribute.Bool(err != nil))

		duration := time.Since(start)
		ir.saveDuration.Record(ctx, duration.Milliseconds(), attributes...)

		if err != nil {
			span.RecordError(err)
		}

		span.End()
	}()

	err = ir.repository.Save(ctx, root)

	return
}

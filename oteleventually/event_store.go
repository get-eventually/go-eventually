package oteleventually

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/trace"

	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/version"
)

// Attribute keys used by the InstrumentedEventStore instrumentation.
const (
	EventStreamIDKey              attribute.Key = "event_stream.id"
	EventStreamVersionSelectorKey attribute.Key = "event_stream.select_from_version"
	EventStreamExpectedVersionKey attribute.Key = "event_stream.expected_version"
	EventStoreNumEventsKey        attribute.Key = "event_store.num_events"
)

var _ event.Store = &InstrumentedEventStore{}

// InstrumentedEventStore is a wrapper type over an event.Store
// instance to provide instrumentation, in the form of metrics and traces
// using OpenTelemetry.
//
// Use NewInstrumentedEventStore for constructing a new instance of this type.
type InstrumentedEventStore struct {
	eventStore event.Store

	tracer         trace.Tracer
	streamDuration syncint64.Histogram
	appendDuration syncint64.Histogram
}

func (ies *InstrumentedEventStore) registerMetrics(meter metric.Meter) error {
	var err error

	if ies.streamDuration, err = meter.SyncInt64().Histogram(
		"eventually.event_store.stream.duration.milliseconds",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("Duration in milliseconds of event.Store.Stream operations performed."),
	); err != nil {
		return fmt.Errorf("oteleventually.InstrumentedEventStore: failed to register metric: %w", err)
	}

	if ies.appendDuration, err = meter.SyncInt64().Histogram(
		"eventually.event_store.append.duration.milliseconds",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("Duration in milliseconds of event.Store.Append operations performed."),
	); err != nil {
		return fmt.Errorf("oteleventually.InstrumentedEventStore: failed to register metric: %w", err)
	}

	return nil
}

// NewInstrumentedEventStore returns a wrapper type to provide OpenTelemetry
// instrumentation (metrics and traces) around an event.Store.
//
// An error is returned if metrics could not be registered.
func NewInstrumentedEventStore(eventStore event.Store, options ...Option) (*InstrumentedEventStore, error) {
	cfg := newConfig(options...)

	ies := &InstrumentedEventStore{
		eventStore: eventStore,
		tracer:     cfg.tracer(),
	}

	if err := ies.registerMetrics(cfg.meter()); err != nil {
		return nil, err
	}

	return ies, nil
}

// Stream calls the wrapped event.Store.Stream method and records metrics and traces around it.
func (ies *InstrumentedEventStore) Stream(
	ctx context.Context,
	stream event.StreamWrite,
	id event.StreamID,
	selector version.Selector,
) (err error) {
	attributes := []attribute.KeyValue{
		EventStreamIDKey.String(string(id)),
		EventStreamVersionSelectorKey.Int64(int64(selector.From)),
	}

	ctx, span := ies.tracer.Start(ctx, "event.Store.Stream", trace.WithAttributes(attributes...))
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		ies.streamDuration.Record(ctx, duration.Milliseconds(), ErrorAttribute.Bool(err != nil))

		if err != nil {
			span.RecordError(err)
		}

		span.End()
	}()

	err = ies.eventStore.Stream(ctx, stream, id, selector)

	return
}

// Append calls the wrapped event.Store.Append method and records metrics and traces around it.
func (ies *InstrumentedEventStore) Append(
	ctx context.Context,
	id event.StreamID,
	expected version.Check,
	events ...event.Envelope,
) (newVersion version.Version, err error) {
	expectedVersion := int64(-1)
	if v, ok := expected.(version.CheckExact); ok {
		expectedVersion = int64(v)
	}

	attributes := []attribute.KeyValue{
		EventStreamIDKey.String(string(id)),
		EventStreamExpectedVersionKey.Int64(expectedVersion),
		EventStoreNumEventsKey.Int(len(events)),
	}

	ctx, span := ies.tracer.Start(ctx, "event.Store.Append", trace.WithAttributes(attributes...))
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		ies.streamDuration.Record(ctx, duration.Milliseconds(), ErrorAttribute.Bool(err != nil))

		if err != nil {
			span.RecordError(err)
		}

		span.End()
	}()

	newVersion, err = ies.eventStore.Append(ctx, id, expected, events...)

	return
}

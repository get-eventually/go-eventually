package opentelemetry

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/eventually-rs/eventually-go/extension/opentelemetry"

// Names of the OpenTelemetry spans created by the package.
const (
	StreamSpanName = "EventStore.Stream"
	AppendSpanName = "EventStore.Append"
)

var (
	// StreamNameLabel is the label identifier that contains the Stream name,
	// or Stream instance id, when using an eventstore.Instanced.
	StreamNameLabel = label.Key("eventually.event_store.stream.name")

	// StreamTypeLabel is the label identifier that contains the Stream type,
	// when using an eventstore.Typed.
	StreamTypeLabel = label.Key("eventually.event_store.stream.type")

	// StreamFromLabel is the label identifier that contains the version or
	// sequence number lower bound used for Stream calls.
	StreamFromLabel = label.Key("eventually.event_store.stream.from")

	// VersionCheckLabel is the label identifier that contains the expected
	// version provided when using Append to add new events to the Event Store.
	VersionCheckLabel = label.Key("eventually.event_store.append.version.check")

	// VersionNewLabel is the label identifier that contains the new version
	// returned by the Event Store on Append calls.
	VersionNewLabel = label.Key("eventually.event_store.append.version.new")
)

var _ eventstore.Store = EventStoreWrapper{}

// EventStoreWrapper is a wrapper to provide OpenTelemetry instrumentation
// for eventstore.Store compatible implementations, and compatible
// with the same interface to be used seamlessly in your pre-existing code.
//
// Use WrapEventStore to create new instance of this type.
type EventStoreWrapper struct {
	eventstore.Store
	tracer trace.Tracer
}

// WrapEventStore creates a new EventStoreWrapper instance, using the provided
// TracerProvider to collect metrics.
func WrapEventStore(es eventstore.Store, tracerProvider trace.TracerProvider) EventStoreWrapper {
	return EventStoreWrapper{
		Store:  es,
		tracer: tracerProvider.Tracer(instrumentationName),
	}
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
		Typed:      tsw,
		streamType: typ,
		tracer:     sw.tracer,
	}, nil
}

type typedEventStoreWrapper struct {
	eventstore.Typed
	streamType string
	tracer     trace.Tracer
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
	return instancedEventStoreWrapper{
		Instanced:  tsw.Typed.Instance(id),
		streamType: tsw.streamType,
		streamName: id,
		tracer:     tsw.tracer,
	}
}

type instancedEventStoreWrapper struct {
	eventstore.Instanced
	streamType string
	streamName string
	tracer     trace.Tracer
}

func (isw instancedEventStoreWrapper) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
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

func (isw instancedEventStoreWrapper) Append(ctx context.Context, version int64, events ...eventually.Event) (int64, error) {
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

	return newVersion, err
}

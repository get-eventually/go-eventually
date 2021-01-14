package opentelemetry

import (
	"context"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/eventstore"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
)

var (
	StreamNameLabel   = label.Key("eventually.event_store.stream.name")
	StreamTypeLabel   = label.Key("eventually.event_store.stream.type")
	StreamFromLabel   = label.Key("eventually.event_store.stream.from")
	VersionCheckLabel = label.Key("eventually.event_store.append.version.check")
	VersionNewLabel   = label.Key("eventually.event_store.append.version.new")
)

var _ eventstore.Store = EventStoreWrapper{}

type EventStoreWrapper struct {
	eventstore.Store
	tracer trace.Tracer
}

func WrapEventStore(es eventstore.Store, tracerProvider trace.TracerProvider) EventStoreWrapper {
	return EventStoreWrapper{
		Store:  es,
		tracer: tracerProvider.Tracer("github.com/eventually-rs/eventually-go"),
	}
}

func (sw EventStoreWrapper) Stream(ctx context.Context, es eventstore.EventStream, from int64) error {
	ctx, span := sw.tracer.Start(ctx, "EventStore.Stream", trace.WithAttributes(
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

// func (sw EventStoreWrapper) Subscribe(ctx context.Context, es eventstore.EventStream) error {
// 	ctx, span := sw.tracer.Start(ctx, "EventStore.Subscribe", trace.WithAttributes(
// 		StreamTypeLabel.String("*"),
// 	))
// 	defer span.End()

// 	err := sw.Store.Subscribe(ctx, es)
// 	if err != nil {
// 		span.RecordError(err)
// 	}

// 	return err
// }

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
	ctx, span := tsw.tracer.Start(ctx, "EventStore.Stream", trace.WithAttributes(
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
	ctx, span := isw.tracer.Start(ctx, "EventStore.Stream", trace.WithAttributes(
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
	ctx, span := isw.tracer.Start(ctx, "EventStore.Append", trace.WithAttributes(
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

package eventstore

import (
	"context"

	"github.com/get-eventually/go-eventually"
	"github.com/get-eventually/go-eventually/eventstore/stream"
)

type metadataContextKey struct{}

// ContextMetadata extends a provided context.Context instance with the provided
// eventually.Metadata map. When using the ContextAware eventstore.Appender extension,
// the provided Metadata map will be applied to all events passed during an Append() method call.
func ContextMetadata(ctx context.Context, metadata eventually.Metadata) context.Context {
	ctxMetadata := ctx.Value(metadataContextKey{})
	if ctxMetadata == nil {
		return context.WithValue(ctx, metadataContextKey{}, metadata)
	}

	// If metadata has been defined in the context already, then merge the new values
	// with the already-existing metadata map.
	ctxMetadata.(eventually.Metadata).Merge(metadata)

	return ctx
}

// ContextAware is an eventstore.Appender extension that uses the eventually.Metadata map
// provided in the context.Context used during an Append() call to extend the metadata
// of each event being appended.
//
// Use ContextMetadata in conjunction with this type to make use of this feature.
type ContextAware struct {
	Appender
}

// NewContextAware extends the provided eventstore.Appender instance with a ContextAware version.
func NewContextAware(appender Appender) ContextAware {
	return ContextAware{Appender: appender}
}

// Append applies the eventually.Metadata map from the context.Context to all the events specified, if such
// map has been provided using eventstore.ContextMetadata.
//
// The extended events are then appended to the Event Store using the base eventstore.Appender instance
// provided during initialization.
func (ca ContextAware) Append(
	ctx context.Context,
	id stream.ID,
	versionCheck VersionCheck,
	events ...eventually.Event,
) (int64, error) {
	metadata, ok := ctx.Value(metadataContextKey{}).(eventually.Metadata)
	if !ok {
		return ca.Appender.Append(ctx, id, versionCheck, events...)
	}

	newEvents := make([]eventually.Event, 0, len(events))

	for _, event := range events {
		event.Metadata = event.Metadata.Merge(metadata)
		newEvents = append(newEvents, event)
	}

	return ca.Appender.Append(ctx, id, versionCheck, newEvents...)
}

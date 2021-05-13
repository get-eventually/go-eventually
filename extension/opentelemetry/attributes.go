package opentelemetry

import "go.opentelemetry.io/otel/attribute"

// Names of the OpenTelemetry spans created by the package.
const (
	StreamSpanName       = "EventStore.Stream"
	StreamByTypeSpanName = "EventStore.StreamByType"
	StreamAllSpanName    = "EventStore.StreamAll"
	AppendSpanName       = "EventStore.Append"
	ApplierSpanName      = "Projection.Applier"
)

// Metrics exported by this package.
const (
	AppendMetric          = "eventually.eventstore.append"
	ProjectionApplyMetric = "eventually.projection.apply.duration.ms"
)

var (
	// StreamNameAttribute is the attribute identifier that contains the Stream name,
	// or Stream instance id, when using an eventstore.Instanced.
	StreamNameAttribute = attribute.Key("stream.name")

	// StreamTypeAttribute is the attribute identifier that contains the Stream type,
	// when using an eventstore.Typed.
	StreamTypeAttribute = attribute.Key("stream.type")

	// EventTypeAttribute is the attribute identifier that contains the type of an Event.
	EventTypeAttribute = attribute.Key("event.type")

	// EventVersionAttribute is the attribute identifier that contains the version of an Event.
	EventVersionAttribute = attribute.Key("event.version")

	// SelectFromAttribute is the attribute identifier that contains the version or
	// sequence number lower bound used for Stream calls.
	SelectFromAttribute = attribute.Key("select.from")

	// VersionCheckAttribute is the attribute identifier that contains the expected
	// version provided when using Append to add new events to the Event Store.
	VersionCheckAttribute = attribute.Key("append.version.check")

	// VersionNewAttribute is the attribute identifier that contains the new version
	// returned by the Event Store on Append calls.
	VersionNewAttribute = attribute.Key("append.version.new")

	// ProjectionNameAttribute is the attribute identifier that contains the name
	// of a specific Projection.
	ProjectionNameAttribute = attribute.Key("projection.name")

	// SubscriptionNameAttribute is the attribute identifier that contains
	// the name of a subscription.
	SubscriptionNameAttribute = attribute.Key("subscrption.name")
)

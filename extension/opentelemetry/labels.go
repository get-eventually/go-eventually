package opentelemetry

import "go.opentelemetry.io/otel/label"

// Names of the OpenTelemetry spans created by the package.
const (
	StreamSpanName  = "EventStore.Stream"
	AppendSpanName  = "EventStore.Append"
	ApplierSpanName = "Projection.Applier"
)

// Metrics exported by this package.
const (
	AppendMetric          = "eventually.eventstore.append"
	ProjectionApplyMetric = "eventually.projection.apply.duration.ms"
)

var (
	// StreamNameLabel is the label identifier that contains the Stream name,
	// or Stream instance id, when using an eventstore.Instanced.
	StreamNameLabel = label.Key("stream.name")

	// StreamTypeLabel is the label identifier that contains the Stream type,
	// when using an eventstore.Typed.
	StreamTypeLabel = label.Key("stream.type")

	// EventTypeLabel is the label identifier that contains the type of an Event.
	EventTypeLabel = label.Key("event.type")

	// EventVersionLabel is the label identifier that contains the version of an Event.
	EventVersionLabel = label.Key("event.version")

	// StreamFromLabel is the label identifier that contains the version or
	// sequence number lower bound used for Stream calls.
	StreamFromLabel = label.Key("stream.from")

	// VersionCheckLabel is the label identifier that contains the expected
	// version provided when using Append to add new events to the Event Store.
	VersionCheckLabel = label.Key("append.version.check")

	// VersionNewLabel is the label identifier that contains the new version
	// returned by the Event Store on Append calls.
	VersionNewLabel = label.Key("append.version.new")

	// ProjectionNameLabel is the label identifier that contains the name
	// of a specific Projection.
	ProjectionNameLabel = label.Key("projection.name")

	// SubscriptionNameLabel is the label identifier that contains
	// the name of a subscription
	SubscriptionNameLabel = label.Key("subscrption.name")
)

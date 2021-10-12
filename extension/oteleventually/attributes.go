package oteleventually

import "go.opentelemetry.io/otel/attribute"

var (
	// ErrorAttribute is used with a metric when an error is recorded.
	ErrorAttribute = attribute.Key("error")

	// StreamTargetAttribute is the attribute identifier that contains the
	// stream target value used for EventStore.Stream calls.
	StreamTargetAttribute = attribute.Key("stream.target")

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

	// EventSequenceNumberAttribute is the attribute identifier that contains the sequence number of an Event.
	EventSequenceNumberAttribute = attribute.Key("event.sequence_number")

	// SelectFromAttribute is the attribute identifier that contains the version or
	// sequence number lower bound used for Stream calls.
	SelectFromAttribute = attribute.Key("select.from")

	// VersionCheckAttribute is the attribute identifier that contains the expected
	// version provided when using Append to add new events to the Event Store.
	VersionCheckAttribute = attribute.Key("version.check")

	// VersionNewAttribute is the attribute identifier that contains the new version
	// returned by the Event Store on Append calls.
	VersionNewAttribute = attribute.Key("version.new")

	// ProcessorNameAttribute is the attribute identifier that contains the name
	// of a specific Projection.
	ProcessorNameAttribute = attribute.Key("processor.name")

	// SubscriptionNameAttribute is the attribute identifier that contains
	// the name of a subscription.
	SubscriptionNameAttribute = attribute.Key("subscrption.name")
)

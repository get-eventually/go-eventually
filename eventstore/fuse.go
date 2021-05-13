package eventstore

// FusedAppendStreamer is a convenience type to fuse
// multiple Event Store interfaces where you might need to extend
// the functionality of the Store only partially.
//
// E.g. You might want to extend the functionality of the Append() method,
// but keep the Streamer methods the same.
//
// If the extension wrapper does not support
// the Streamer interface, you cannot use the extension wrapper instance as an
// Event Store in certain cases (e.g. the Aggregate Repository).
//
// Using a FusedAppendStreamer instance you can fuse both instances
// together, and use it with the rest of the library ecosystem.
type FusedAppendStreamer struct {
	Appender
	Streamer
}

// FusedStreamSubscriber is a convenience type to fuse the Streamer and
// Subscriber interfaces of an Event Store, where you might need to extend
// the functionality of such interfaces only partially.
//
// Check out FusedAppendStreamer for more information.
type FusedStreamSubscriber struct {
	Streamer
	Subscriber
}

package snapshot

import "time"

// Policy represents the behavior of the Snapshot functionality,
// advising on the frequency of the snapshots to take.
//
// Choose the best Policy among the ones provided in this package, considering
// your needs and the rate of updates of the Aggregate Root you're trying to optimize.
type Policy interface {
	ShouldRecord(version int64) bool
	Record(version int64)
}

// NeverPolicy is a Snapshot Policy that never signals to take snapshots
// when queried.
type NeverPolicy struct{}

// ShouldRecord always returns false.
func (NeverPolicy) ShouldRecord(version int64) bool { return false }

// Record is a no-op.
func (NeverPolicy) Record(version int64) {}

// AlwaysPolicy is a Snapshot Policy that always signals to take snapshots
// when queried.
type AlwaysPolicy struct{}

// ShouldRecord always returns true.
func (AlwaysPolicy) ShouldRecord(version int64) bool { return true }

// Record is a no-op.
func (AlwaysPolicy) Record(version int64) {}

// AtFixedIntervalsPolicy is a Snapshot Policy that signals to take snapshots
// at a fixed, specified time interval (e.g. every 1 hour, etc.)
//
// Please note: the time interval is calculated from the start of the application,
// not from the last Snapshot inserted in the Snapshot store. This is important
// to keep in mind while debugging your application and the snapshot behavior.
type AtFixedIntervalsPolicy struct {
	interval time.Duration
	lastTime time.Time
}

// NewAtFixedIntervalsPolicy creates an AtFixedIntervalsPolicy instance
// with the specified time interval for Snapshot recordings.
func NewAtFixedIntervalsPolicy(interval time.Duration) *AtFixedIntervalsPolicy {
	return &AtFixedIntervalsPolicy{}
}

// ShouldRecord returns true on the first query, then after every interval
// specified during construction.
func (p *AtFixedIntervalsPolicy) ShouldRecord(version int64) bool {
	return time.Since(p.lastTime) >= p.interval
}

// Record updates the internal state of the Policy with the current timestamp.
func (p *AtFixedIntervalsPolicy) Record(version int64) {
	p.lastTime = time.Now()
}

// EveryVersionIncrementPolicy is a Snapshot Policy that signals to take
// snapshots every version increment specified by this value.
//
// If the number used is EveryVersionIncrementPolicy(10), it means this policy
// will signal to record a snapshot at version 10, 20, 30 and so on.
type EveryVersionIncrementPolicy int64

// ShouldRecord returns true when the current version modulo the increment
// specified in this policy equals to zero.
func (v EveryVersionIncrementPolicy) ShouldRecord(version int64) bool {
	return version%int64(v) == 0
}

// Record is a no-op, as the policy uses a stateless function.
func (EveryVersionIncrementPolicy) Record(version int64) {}

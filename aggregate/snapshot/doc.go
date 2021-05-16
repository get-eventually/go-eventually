// Package snapshot provides support for Aggregate Root snapshots, useful
// where the size of your Aggregate Roots is expected to considerably
// grow in size and number of events.
//
// Snapshots are used by an Event-sourced Aggregate Repository as an optimization
// technique to speed up the Aggregate state rehydration process, by saving
// the state of the Aggregate Root at a particular point in time in a durable store.
package snapshot

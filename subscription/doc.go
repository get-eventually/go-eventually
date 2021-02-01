// Package subscription contains Event Subscription implementations to
// listens and process Events coming from an Event Store, such as
// running Projections.
//
// Choose the Subscription type that is suited to your Event processor.
// Catch-up Subscriptions are the most commonly used type of Subscription,
// especially for Projections or Process Managers.
//
// Volatile Subscriptions might be used for volatile Projections,
// e.g. when process restarts should erase the previous Projection value,
// typical for instance summaries or metrics recording, for example.
package subscription

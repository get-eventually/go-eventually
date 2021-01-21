// Package eventually contains types and abstraction to allow you to write
// Event-sourced application, without having to take care of the infrastructure
// setup necessary to run such an architecture.
//
// The library contains multiple packages, you might want to start from `aggregate`
// to implement your Aggregate types, and `command` to implement the Command Handlers
// to interact with or update your Aggregates.
//
// `query` and `projection` allows you to implement Domain Queries and Read Models.
package eventually

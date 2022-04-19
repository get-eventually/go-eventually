# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this
project adheres to [Semantic Versioning](https://semver.org/).


## [Unreleased]
### Added
- Usage of Go workspaces for local development.
- New `core/message` package for defining messages.
- `core/serde` package for serialization and deserialization of types.
- `serdes` module using `core/serde` with some common protocol implementations: **Protobuf**, **ProtoJSON** and **JSON**.
- `postgres.AggregateRepository` implementation to load/save Aggregates directly, and still saving recorded Domain Events in a separate table (`events`).

### Changed
- `aggregate` package uses Go generics for `aggregate.Repository` and `aggregate.Root` interfaces.
- `eventually.Payload` is now `message.Message`.
- `eventually.Message` is now `message.Envelope`.
- `eventstore.Event` is now `event.Persisted`.
- `eventstore.Store` is now `event.Store`.
- `command.Command[T]` is now using `message.Message[T]`.
- `command.Handler` is now generic over its `command.Command` input.
- `scenario` package is now under `core/test/scenario`.
- `scenario.CommandHandler` now uses generics for command and command handler assertion.
- `postgres` module now uses `pgx` to handle connection with the PostgreSQL database, instead of `database/sql`.
- `postgres.EventStore` uses `serde.Serializer` interface to serialize/deserialize Domain Events to `[]byte`.

### Removed
- `SequenceNumber` from the `event.Persisted` struct (was previously `eventstore.Event`).
- `eventstore.SequenceNumberGetter`, to follow the previous `SequenceNumber` removal.
- `command.Dispatcher` interface, as implementing it with generics is currently not possible.

## [Pre-v0.2.0 unreleased changes]
### Changed
- Add `logger.Logger` to `command.ErrorRecorder` to report errors when appending Command failures to the Event Store.
- `command.ErrorRecorder` must be passed by reference to implement `command.Handler` interface now (size of the struct increased).

### Removed
- Remove the `events` field from `oteleventually.InstrumentedEventStore` due to the potential size of the field and issues with exporting the trace (which wouldn't fit an UDP packet).
- Remove the `event` field from `oteleventually.InstrumentedProjection`.

## [v0.1.0-alpha.4]
### Added
- `X-Eventually-TraceId` and `X-Eventually-SpanId` metadata keys are recorded when using `oteleventually.InstrumentedEventStore.Append`.
- Add `eventstore.ContextAware` and `eventstore.ContextMetadata` to set some Metadata in the context to be applied to all Domain Events appended to the Event Store.

### Changed
- `postgres.Serializer` and `postgres.Deserializer` use `stream.ID` for the mapping function.
- Update `go.opentelemetry.io/otel` to `v1.2.0`
- Update `go.opentelemetry.io/otel/metric` to `v0.25.0`

## [v0.1.0-alpha.3]
### Added
- Testcase for the Event Store testing suite to assert that `eventstore.Appender.Append` returns `eventstore.ErrConflict`.
- `postgres.EventStore.Append` returns `eventstore.ErrConflict` in case of conflict now.

### Changed
- Metric types in `oteleventually` have been adapted to the latest `v0.24.0` version.
- `eventstore.ErrConflict` has been renamed to `eventstore.ConflictError`.

## [v0.1.0-alpha.2]
### Added
- An option to override Event appending logic in Postgres EventStore implementation.
- `postgres.Serde` interface to support more serialization formats.

### Changed
- Existing `Event-Id` value in Event Metadata does not get overwritten in correlation.EventStoreWrapper.
- `postgres.EventStore` now uses the `Serde` interface for serializing to and deserializing from byte array.
- `postgres.Registry` is now called `postgres.JSONRegistry` and implements thenew `postgres.Serde` interface.
- `CaptureErrors` in `command.ErrorRecorder` is now a function (`ShouldCaptureError`), to allow for a more flexible capture strategy.

## [v0.1.0-alpha.1]

A lot of changes have happened here, a lot of different API design iterations and stuff. All of which, I diligently forgot to keep track of...

Sorry :)

<!-- @formatter:off -->
[Unreleased]: https://github.com/get-eventually/go-eventually/compare/eb0deb0..HEAD
[Pre-v0.2.0 unreleased changes]: https://github.com/get-eventually/go-eventually/compare/eb0deb0..HEAD
[v0.1.0-alpha.4]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.4..v0.1.0-alpha.3
[v0.1.0-alpha.3]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.2..v0.1.0-alpha.3
[v0.1.0-alpha.2]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.1..v0.1.0-alpha.2
[v0.1.0-alpha.1]: https://github.com/get-eventually/go-eventually/compare/8bb9190..v0.1.0-alpha.1

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this
project adheres to [Semantic Versioning](https://semver.org/).

## [0.2.0-alpha.2](https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.2...v0.2.0-alpha.2) (2023-12-18)


### Features

* **command:** add logger.Logger to ErrorReporter ([#76](https://github.com/get-eventually/go-eventually/issues/76)) ([1e02257](https://github.com/get-eventually/go-eventually/commit/1e022578e2452f6a3e6a18bcded43be07eebf291))
* **core/command:** add some helper functions and documentation ([#86](https://github.com/get-eventually/go-eventually/issues/86)) ([ee1f26e](https://github.com/get-eventually/go-eventually/commit/ee1f26e6a2924b6ad2ecff5930918f8ec31516c2))
* **eventstore:** add ContextAware and ContextMetadata propagator ([#72](https://github.com/get-eventually/go-eventually/issues/72)) ([6d8e3b2](https://github.com/get-eventually/go-eventually/commit/6d8e3b23a4c32af278122d5fe79ccda7e63188c0))
* **github:** add github actions dependabot ([#102](https://github.com/get-eventually/go-eventually/issues/102)) ([fd1c6d5](https://github.com/get-eventually/go-eventually/commit/fd1c6d5e9729af4fcf9f42e35b7c43a35877fb73))
* **github:** enable dependabot weekly reports ([#95](https://github.com/get-eventually/go-eventually/issues/95)) ([8594cd1](https://github.com/get-eventually/go-eventually/commit/8594cd1ad0cdab60861e033a4038bc52b980627c))
* implement Firestore event.Store interface ([#136](https://github.com/get-eventually/go-eventually/issues/136)) ([5e1c10c](https://github.com/get-eventually/go-eventually/commit/5e1c10c04d5a51b89da7ba146665882fdfeba237))
* **opentelemetry:** add aggregate.version to the Repository.Get span ([#142](https://github.com/get-eventually/go-eventually/issues/142)) ([e204ba9](https://github.com/get-eventually/go-eventually/commit/e204ba9f10ae6c1558b1a169b70b496026796034))
* **oteleventually:** add trace id and span id to events metadata during Append ([#71](https://github.com/get-eventually/go-eventually/issues/71)) ([70f9505](https://github.com/get-eventually/go-eventually/commit/70f9505fe9c2771c8a7a4bfa72389c034d88baa5))
* **postgres:** EventStore.Append returns eventstore.ErrConflict in case of conflict error ([#67](https://github.com/get-eventually/go-eventually/issues/67)) ([e1617a9](https://github.com/get-eventually/go-eventually/commit/e1617a97d6a543e728f7e188cfeeaea3f3d3e933))
* **postgres:** update to pgx/v5 and refactor RunMigrations ([#137](https://github.com/get-eventually/go-eventually/issues/137)) ([a74cc5d](https://github.com/get-eventually/go-eventually/commit/a74cc5d818ba390bc3b0ec19cee94a9c8d9de4f4))
* **README:** add matrix chat link ([#105](https://github.com/get-eventually/go-eventually/issues/105)) ([3594928](https://github.com/get-eventually/go-eventually/commit/3594928062546d54dd8eb80783de86d607e48784))


### Bug Fixes

* golangci-lint linter execution ([#88](https://github.com/get-eventually/go-eventually/issues/88)) ([bff3e52](https://github.com/get-eventually/go-eventually/commit/bff3e5219f413465268811a6f7296a5f21ea122a))
* **oteleventually:** metric name for repository.save ([#84](https://github.com/get-eventually/go-eventually/issues/84)) ([a0bb5be](https://github.com/get-eventually/go-eventually/commit/a0bb5be3e485256b438050fd6557dddb9800ed36))
* **oteleventually:** remove 'event' field from InstrumentedProjection ([#78](https://github.com/get-eventually/go-eventually/issues/78)) ([75e48fd](https://github.com/get-eventually/go-eventually/commit/75e48fd2b585a33dd2cdac7fcad82c1f113c269d))
* **oteleventually:** remove 'events' field from InstrumentedEventStore ([#77](https://github.com/get-eventually/go-eventually/issues/77)) ([d1725e3](https://github.com/get-eventually/go-eventually/commit/d1725e35930a7e6769e99c3aa36e69ebec123faa))
* **postgres:** use eventually_schema_migrations table for migrations ([#87](https://github.com/get-eventually/go-eventually/issues/87)) ([4886b08](https://github.com/get-eventually/go-eventually/commit/4886b082d33db4741832bba12623bfd668790913))
* **postgres:** use pgxpool.Pool instead of pgx.Conn for db communication ([#83](https://github.com/get-eventually/go-eventually/issues/83)) ([076e4e8](https://github.com/get-eventually/go-eventually/commit/076e4e86145407b81caa03bc900babe61a584917))
* **postgres:** use SERIALIZABLE tx isolation level for AggregateRepository.Save ([#139](https://github.com/get-eventually/go-eventually/issues/139)) ([0cde29d](https://github.com/get-eventually/go-eventually/commit/0cde29d98de6a1cb38ec250d9dd822af6a5de477))
* **release-please-config:** add / as tag separator ([6ce8d8c](https://github.com/get-eventually/go-eventually/commit/6ce8d8c39d912276f99512b17cf452384979bd90))

## [Unreleased]

### Added

- Usage of Go workspaces for local development.
- New `core/message` package for defining messages.
- `core/serde` package for serialization and deserialization of types.
- `serdes` module using `core/serde` with some common protocol implementations: **Protobuf**, **ProtoJSON** and **JSON**.
- `postgres.AggregateRepository` implementation to load/save Aggregates directly, and still saving recorded Domain Events in a separate table (`events`).
- `oteleventually.InstrumentedRepository` provides an `aggregate.Repository` instrumentation.
- New `scenario.AggregateRoot` API to BDD-like test scenario on an `aggregate.Root` instance.

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
- `postgres` module now uses `pgx` and `pgxpool` to handle connection with the PostgreSQL database, instead of `database/sql`.
- `postgres.EventStore` uses `serde.Serializer` interface to serialize/deserialize Domain Events to `[]byte`.
- `oteleventually.InstrumentedEventStore` is now adapted to the new `event.Store` interface.

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

[unreleased]: https://github.com/get-eventually/go-eventually/compare/eb0deb0..HEAD
[pre-v0.2.0 unreleased changes]: https://github.com/get-eventually/go-eventually/compare/eb0deb0..HEAD
[v0.1.0-alpha.4]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.4..v0.1.0-alpha.3
[v0.1.0-alpha.3]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.2..v0.1.0-alpha.3
[v0.1.0-alpha.2]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.1..v0.1.0-alpha.2
[v0.1.0-alpha.1]: https://github.com/get-eventually/go-eventually/compare/8bb9190..v0.1.0-alpha.1

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this
project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]
### Added
- ...
### Changed
- Add `logger.Logger` to `command.ErrorRecorder` to report errors when appending Command failures to the Event Store.
- `command.ErrorRecorder` must be passed by reference to implement `command.Handler` interface now (size of the struct increased).

### Deprecated
- ...

### Removed
- ...

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
[Unreleased]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.4..HEAD
[v0.1.0-alpha.4]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.4..v0.1.0-alpha.3
[v0.1.0-alpha.3]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.2..v0.1.0-alpha.3
[v0.1.0-alpha.2]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.1..v0.1.0-alpha.2
[v0.1.0-alpha.1]: https://github.com/get-eventually/go-eventually/compare/8bb9190..v0.1.0-alpha.1

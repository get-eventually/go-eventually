# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this
project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]
### Added
- ...

### Changed
- `projection` package has been removed, the types have been moved:
    - `projection.Applier` is now `event.Processor`,
    - `projection.Runner` is now `event.ProcessorRunner`.
- `eventstore` package has been removed, and type implementations and definitions moved to the `event` package instead.
- `stream` package has been removed, the `stream.ID` type is now under `event.StreamID`.

### Deprecated
- ...

### Removed
- `subscription` package has been removed.
- First-level support from the library to stream from multiple targets (`stream.ByType`, `stream.All`, etc.) has been removed. The `event.Streamer` interface now only targets a _single Event Stream_; for everything else, use an `event.Subscription` implementation instead.

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
[Unreleased]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.2..HEAD
[v0.1.0-alpha.2]: https://github.com/get-eventually/go-eventually/compare/v0.1.0-alpha.1..v0.1.0-alpha.2
[v0.1.0-alpha.1]: https://github.com/get-eventually/go-eventually/compare/8bb9190..v0.1.0-alpha.1

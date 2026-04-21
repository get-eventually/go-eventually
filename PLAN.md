# go-eventually — Improvement Plan

This document captures the agreed-upon plan to unstick the `go-eventually`
library from its release limbo and bring the codebase, release pipeline, and
developer experience up to current standards.

## Context

`go-eventually` is a Go library for Domain-driven Design, Event Sourcing and
CQRS. It has a strong architectural spine (clean interface segregation,
disciplined use of generics, BDD-style test scaffolding) but has been
effectively paused since `v0.3.0-prerelease.5` (~18 months). The release
pipeline is misconfigured for a Go library, the module layout contradicts its
own tagging scheme, and the codebase has accumulated a handful of concrete
bugs and documentation drift.

## Locked-in decisions

- **Module layout**: single-module (root `go.mod` only). Subdir-style tags
  (`postgres/v0.2.1`, `firestore/v0.2.1`, …) will be deleted.
- **Next release target**: `v0.4.0` to end the prerelease limbo.
- **Release automation**: `release-please`.
- **CI matrix**: Go 1.24 × Linux / macOS / Windows.
- **Selected breaking changes for v0.4.0**:
  - Drop the `firestore/` adapter (unused, not bringing value).
  - Replace channel-based event streaming with `iter.Seq2`.
  - Make `message.Metadata.With`/`Merge` copy-on-write.
  - Fix `TrackingStore` to wrap a full `event.Store`, not just `Appender`.
- **Non-goals**: `aggregate.Root` sealing via `BaseRoot` embedding is
  intentional and will be **documented**, not changed.
- **Ordering**: correctness → release automation → CI → DX → breaking changes.

## Impact of dropping Firestore

- **Code removed**: entire `firestore/` directory
  (`event_store.go`, `event_store_test.go`, `doc.go`).
- **Dependencies removed** from `go.mod`:
  - `cloud.google.com/go/firestore`
  - `github.com/testcontainers/testcontainers-go/modules/gcloud`
  - `google.golang.org/api`
  - `google.golang.org/genproto`
  - Transitive: `cloud.google.com/go`, `cloud.google.com/go/auth`,
    `cloud.google.com/go/longrunning`, `cloud.google.com/go/compute/metadata`,
    `github.com/apache/arrow/go/v15`, `github.com/googleapis/*`,
    `github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp`,
    `github.com/google/s2a-go`, etc.
- **Consequence**: meaningful `go.sum` shrinkage and CVE-surface reduction for
  every consumer of the library.
- **Bugs we no longer need to fix**: Firestore metadata deserialization
  (`firestore/event_store.go:79`), public-field/constructor mismatch,
  hardcoded collection names, missing Firestore `AggregateRepository`,
  ctx-send deadlock in Firestore `Stream`.

---

## Phase 1 — Correctness fixes (cut as `v0.3.1`, non-breaking)

Small, targeted patches. None of these change exported signatures.

1. **`opentelemetry/event_store.go:138`** — `Append` records into
   `streamDuration`; change to `appendDuration`.
2. **`postgres/event_store.go:72`** — wrap the lost `err` with `%w`.
3. **`postgres/event_store.go:56`** — remove dead
   `errors.Is(err, pgx.ErrNoRows)` branch on `Query` (pgx only returns it
   from `QueryRow`).
4. **`serde/chained.go:37,42`** — fix swapped "first stage"/"second stage"
   labels in `Deserialize`.
5. **Stale/wrong godocs**:
   - `event/store_tracking.go:34` — doc says events don't record the version
     but the code does.
   - `query/query.go:40` — "xquery.Handler" → "query.Handler".
6. **OTel observability**: pair every `span.RecordError(err)` with
   `span.SetStatus(codes.Error, err.Error())` in
   `opentelemetry/event_store.go` and `opentelemetry/repository.go`.
7. **Channel-send cancellation** in `postgres/event_store.go:85` — wrap the
   send with
   `select { case stream <- ...; case <-ctx.Done(): return ctx.Err() }`
   to avoid orphaning the producer goroutine.
8. **Remove ineffective `pgx.Deferrable`** from read-write transactions in
   `postgres/event_store.go:107` and
   `postgres/aggregate_repository.go:119` (pgx ignores `Deferrable` on
   non-read-only transactions).
9. **Fix `BaseRoot` receiver inconsistency** — change `Version()` to a
   pointer receiver (`aggregate/aggregate.go:94`) for consistency with the
   other methods on the same type.
10. **OTel error-message prefix** — change `oteleventually.*` to
    `opentelemetry.*` to match the actual package name.

Tag `v0.3.1` after these land.

> Removing `firestore/` is breaking (import-path change) and is deferred to
> Phase 5 / v0.4.0; it does not belong in a patch release.

## Phase 2 — Release pipeline rebuild

11. **Delete `.goreleaser.yaml`**. goreleaser builds binaries; this library
    has no `main` package.
12. **Rewrite `.github/workflows/release.yml`** around
    `googleapis/release-please-action`:
    - Trigger: `on: push: branches: [main]`.
    - release-please opens a PR that updates `CHANGELOG.md` from Conventional
      Commits.
    - On merge, it cuts the tag and the GitHub Release.
    - Config: `release-please-config.json` with `release-type: go`, seeded
      via `.release-please-manifest.json`.
13. **Tag hygiene** — purge bogus tags from origin:
    - `postgres-v0.2.1`, `postgres/v0.2.1`
    - `firestore-v0.2.1`, `firestore/v0.2.1`
    - `core-v0.2.1`, `core/v0.2.1`
    - `oteleventually-v0.2.1`, `oteleventually/v0.2.1`
    - `serdes-v0.2.1`, `serdes/v0.2.1`
    - `integrationtest-v0.2.1`, `integrationtest/v0.2.1`
    - `eventstore/postgres/v0.1.0-alpha.2..4`

    Keep only root `vX.Y.Z[-*]` tags.
14. **Rewrite `CHANGELOG.md`** — let release-please regenerate; seed only
    with the tail of real history.
15. **Add `RELEASING.md`** documenting:
    - Tag scheme (`vMAJOR.MINOR.PATCH` only, at root).
    - Conventional Commits contract.
    - Pre-v1 SemVer dialect (moving from README).
    - Manual override procedure for emergencies.
16. **Remove `force_push_tag: true`** and the fragile
    `contains(github.ref, 'main')` branch guard (replaced by release-please).

## Phase 3 — CI hardening

17. **Rewrite `.github/workflows/test.yml`**:
    - Matrix: `os: [ubuntu-latest, macos-latest, windows-latest]`, single
      Go 1.24.
    - Unit tests (`go.test.unit` with `-short -race`) run on all three OSes.
    - Integration tests (testcontainers/Postgres only; no more Firestore
      emulator) run only on `ubuntu-latest` as a separate job.
    - Coverage upload only from the Linux integration job.
18. **Bump `go.mod` to `go 1.24`**; align `go.mod`, flake, and CI.
19. **Add `govulncheck`** as a separate job (new `security.yml` or inside
    `lint.yml`).
20. **Replace `magic-nix-cache-action`** with
    `DeterminateSystems/flakehub-cache-action` in
    `.github/actions/nix-setup/action.yaml` (magic-nix-cache was sunset).
21. **Trim `super-linter`** or remove it — it overlaps with `golangci-lint`.
    Keep markdownlint / yamllint as standalone steps if useful.
22. **SonarCloud**: either wire it in after coverage, or delete
    `sonar-project.properties`. Recommendation: delete unless actively used.
23. **Makefile hardening**:
    - Remove `.always-make`; use explicit `.PHONY`.
    - Add `help`, `fmt`, `vet`, `tidy-check`, `lint-fix`, `test-race` targets.
    - Add `-race` to default test flags.

## Phase 4 — Developer experience & governance

24. **Create `CONTRIBUTING.md`** (unblocks the broken README link) with:
    Nix flake setup, how to run tests, commit convention, PR process.
25. **Create `SECURITY.md`** with responsible-disclosure contact.
26. **Create `CODEOWNERS`** mapping paths to maintainers.
27. **Add `.github/PULL_REQUEST_TEMPLATE.md`** and
    `.github/ISSUE_TEMPLATE/{bug_report,feature_request}.md`.
28. **Create `examples/` directory**:
    - `examples/aggregate/` — plain ES aggregate with in-memory store.
    - `examples/postgres/` — using `postgres.EventStore` +
      `AggregateRepository`.
    - `examples/opentelemetry/` — instrumented repository.
    - *(No Firestore example — package removed.)*
29. **Expand `doc.go`** in `aggregate`, `event`, `message`, `command`,
    `query`, `serde`, `version` with:
    - Conceptual overview (2–4 paragraphs).
    - Package-level runnable `Example*` functions.
    - `aggregate/doc.go` must explicitly document the `BaseRoot` embedding
      contract and the `RecordThat` flow — this encodes the deliberate
      sealing design.
30. **`.golangci.yaml` cleanup**:
    - Remove unused `varnamelen` settings block.
    - Set `unparam.check-exported: false` (noisy for library APIs).
    - Inline comments explaining intentional linter choices.
31. **Renovate**: enable the Nix manager so `flake.lock` is auto-bumped.
    Remove the no-op `packageRules` entry.

## Phase 5 — Selective breaking changes (cut as `v0.4.0`)

32. **Drop the `firestore/` package**:
    - Remove directory `firestore/` (all files).
    - Run `go mod tidy` to prune the Firestore dep tree from
      `go.mod` / `go.sum`.
    - Remove the Firestore integration test job from CI.
    - Changelog: `feat!: drop firestore event store adapter` with a migration
      note (only `v0.3.0-prerelease.*` ever referenced it; no stable version
      shipped with Firestore).
    - Remove any Firestore mentions from README / docs / examples.
    - `firestore*` tags already purged in Phase 2.
    - Archive the last Firestore-containing commit SHA in `RELEASING.md`
      for users who want to fork.
33. **`iter.Seq2[event.Persisted, error]` streaming**:
    - Introduce `StreamIter(ctx, Selector) iter.Seq2[event.Persisted, error]`
      on `event.Streamer`.
    - Keep channel-based `Stream` for one release marked `Deprecated:`;
      remove in v0.5.
    - Update `aggregate/event_sourced_repository.go` to use the iterator
      form (removes `errgroup` ceremony and the ctx-send deadlock class).
    - Update remaining adapters: `postgres`, `event/store_inmemory`,
      `event/store_tracking`.
34. **Copy-on-write `message.Metadata.With`/`Merge`**:
    - Return a fresh map; doc updated to match.
    - Internal hot paths that relied on in-place mutation get explicit
      private helpers.
35. **`TrackingStore` wraps full `Store`**:
    - Change `NewTrackingStore` to accept a `Store` (not `Appender`).
    - Embed `Store` so `Stream` is properly promoted.

---

## Execution sketch

- **PR 1** — Phase 1 correctness fixes + seed
  `release-please-manifest.json` at `v0.3.0`. Merge → release-please PR →
  merge → cut `v0.3.1`.
- **PR 2** — Phase 2 remainder: delete goreleaser, swap release workflow,
  add `RELEASING.md`. Tag purge executed once manually (documented there).
- **PR 3** — Phase 3: CI matrix overhaul, `go.mod` floor bump, Makefile
  cleanup.
- **PR 4** — Phase 4: DX / governance files + examples + `doc.go`
  expansions. Can be split by topic if the review surface is too large.
- **PR 5** — **Drop Firestore** (`feat!: remove firestore adapter`).
  Touches `firestore/`, `go.mod`, `go.sum`, CI workflow, README, CHANGELOG.
- **PR 6** — `iter.Seq2` streaming
  (`feat!: replace channel-based event streaming with iterators`).
- **PR 7** — Copy-on-write `Metadata`
  (`feat!: make message.Metadata.With/Merge copy-on-write`).
- **PR 8** — `TrackingStore` fix
  (`feat!: NewTrackingStore accepts a full event.Store`).
- Merging PRs 5–8 produces a release-please PR proposing `v0.4.0` with all
  breaking changes listed in the auto-generated changelog.

## Open question

Any stakeholders currently using the Firestore adapter in production? The
README doesn't advertise it and no stable version ever shipped with it, but
if external users are known, a brief deprecation announcement (e.g. a pinned
issue) before PR 5 is courteous. Otherwise the `feat!` commit and v0.4.0
changelog entry are sufficient.

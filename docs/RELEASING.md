# Releasing go-eventually

`go-eventually` is released through
[release-please](https://github.com/googleapis/release-please). Every push to
`main` is observed by the `Release` workflow
(`.github/workflows/release.yml`), which keeps an open pull request with the
next proposed version bump and an auto-generated changelog entry. Merging that
pull request cuts the tag and the GitHub Release.

## TL;DR

1. Land your changes on `main` via pull requests using
   [Conventional Commits](https://www.conventionalcommits.org/).
2. release-please opens (or refreshes) a `chore(main): release X.Y.Z` pull
   request with the computed version bump and the changelog diff.
3. Review the PR, merge when satisfied.
4. On merge, release-please pushes the `vX.Y.Z` tag and publishes the GitHub
   Release.

No manual tagging. No `git tag -a`. No `goreleaser`.

## Tag scheme

**Only** `vMAJOR.MINOR.PATCH[-PRERELEASE]` tags at the repository root.

## SemVer dialect (pre-v1)

Until the library hits `v1.0.0`, the following dialect is in effect:

- **Breaking changes** (`feat!:`, `fix!:`, or a `BREAKING CHANGE:` footer) bump
  the **minor** version (`0.3.0` → `0.4.0`).
- Everything else — new features (`feat:`), fixes (`fix:`), performance
  improvements (`perf:`), refactors (`refactor:`), docs (`docs:`) — bumps the
  **patch** version (`0.3.0` → `0.3.1`).
- `chore`, `ci`, `build`, `style`, `test` do not appear in the changelog (but
  can still trigger a patch release if they land standalone; avoid if
  possible).

This is encoded in `release-please-config.json` via:

- `bump-minor-pre-major: false` — keep breaking → minor semantics.
- `bump-patch-for-minor-pre-major: false` — do not demote breaking changes to
  patch releases while pre-v1.

After `v1.0.0`, flip `bump-minor-pre-major: true` and drop this section.

## Conventional Commits contract

Commit messages on `main` must follow
[Conventional Commits](https://www.conventionalcommits.org/). The type prefix
decides which section of the changelog the commit lands in, and whether it
triggers a release.

Accepted types:

| Type       | Section in changelog     | Release?           |
| ---------- | ------------------------ | ------------------ |
| `feat`     | Features                 | patch (pre-v1)     |
| `feat!`    | Features (BREAKING)      | **minor** (pre-v1) |
| `fix`      | Bug Fixes                | patch              |
| `perf`     | Performance Improvements | patch              |
| `refactor` | Code Refactoring         | patch              |
| `revert`   | Reverts                  | patch              |
| `docs`     | Documentation            | patch              |
| `deps`     | Dependencies             | patch              |
| `build`    | (hidden)                 | patch              |
| `ci`       | (hidden)                 | patch              |
| `chore`    | (hidden)                 | patch              |
| `style`    | (hidden)                 | patch              |
| `test`     | (hidden)                 | patch              |

Commits that do not match any of the above are ignored by release-please.

**Marking breaking changes.** Append `!` to the type (`feat!:`, `fix!:`) _or_
add a `BREAKING CHANGE:` footer to the commit body. Both work; the `!` form is
preferred for brevity.

**Scopes** are optional and free-form (`feat(postgres): …`, `fix(otel): …`).
Scopes are rendered inline in the changelog but do not affect versioning.

## The release pull request

release-please maintains a single pull request titled
`chore(main): release X.Y.Z`. It:

- updates `docs/CHANGELOG.md` with the accumulated entries since the last
  release;
- updates `.release-please-manifest.json` with the new version.

You review and merge that PR when you are ready to cut a release.
Merging will:

1. Push the `vX.Y.Z` tag.
2. Create a GitHub Release with the release notes.

`pkg.go.dev` will pick up the new version on its next indexing pass (usually
within minutes of the first `go get`).

## Emergency / manual release

If release-please is broken or you need to cut a release outside its normal
flow, the escape hatch is:

1. Ensure `main` is green.
2. Manually tag locally:

   ```sh
   git fetch --tags origin
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

3. Create the GitHub Release manually, copying the changelog entry.
4. Bump `.release-please-manifest.json` to the tagged version in a follow-up
   PR so release-please picks up from the right baseline.

Do not use `git push --force` on tags. Do not re-tag an existing version.

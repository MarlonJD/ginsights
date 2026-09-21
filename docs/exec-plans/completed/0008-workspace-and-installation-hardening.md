# 0008 — Workspace analysis and installation hardening

## Status

Complete.

## Goal

Add an explicit multi-repository workspace mode and make both supported installation paths reproducible from the correct Go module and an immutable Homebrew release source.

## Scope

- Add `--workspace` to `serve`, `build`, and `json`.
- Discover the requested root repository plus nested repositories, including parent-ignored repositories such as the Avia workspace layout.
- Analyze each repository independently, preserve repository identity in aggregate metrics, and render a workspace repository inventory.
- Keep existing single-repository behavior unchanged when `--workspace` is absent.
- Build the shell installer from the cloned module root and cover the behavior with an integration test.
- Replace the mutable Homebrew `main` source with a versioned release tarball and SHA256 once the first release artifact exists.
- Update install and workspace usage documentation.

## Non-goals

- Do not contact GitHub automatically during workspace discovery or core analysis.
- Do not infer or rewrite Git submodule configuration.
- Do not rank contributors across repositories.
- Do not publish a GitHub tag/release or update the external tap without explicit authorization for those external writes.

## Acceptance criteria

- `ginsights serve . --workspace`, `build`, and `json` accept the flag.
- Workspace discovery finds both the root repository and nested `.git` directories or files without following symlinked directory trees.
- Workspace JSON contains per-repository snapshots plus clearly labeled aggregate totals, activity, files, languages, and provenance.
- Workspace HTML identifies itself as a workspace and lists every discovered repository with useful summary metrics.
- A failure in one discovered repository is reported with its path and does not silently corrupt another repository's metrics.
- The source installer builds successfully when invoked from an unrelated Git working directory.
- The Homebrew formula passes syntax and audit checks against an immutable versioned source after release publication.
- Existing single-repository tests continue to pass.

## Verification

```bash
GOCACHE=/tmp/ginsights-go-cache go test ./...
GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .
GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights build . --workspace --out /tmp/ginsights-workspace-report
bash -n scripts/install.sh
ruby -c packaging/homebrew/Formula/ginsights.rb
brew audit --strict --formula packaging/homebrew/Formula/ginsights.rb
```

## Decision log

- 2026-09-21: Keep single-repository mode as the default and require explicit `--workspace` to avoid surprising recursive scans.
- 2026-09-21: Treat each Git root as an independent repository and aggregate derived snapshots rather than flattening histories into a fictional repository.
- 2026-09-21: Discover filesystem Git boundaries directly because Avia intentionally ignores its nested repositories from the parent repository.
- 2026-09-21: Use an immutable tagged release source for Homebrew; the repository currently has no tags or releases, so final SHA verification follows release publication.
- 2026-09-21: Publish the first stable source tag as `v0.1.0`; its GitHub archive SHA256 is `2e3c3458d918ca5eedb3ab4024d27855ba95939581d453274cd065cc00c78b1b`.

## Progress log

- 2026-09-21: Added explicit `--workspace` support to `serve`, `build`, and `json` with filesystem Git-root discovery, per-repository snapshots, aggregate metrics, partial-failure reporting, and workspace HTML inventory.
- 2026-09-21: Prevented parent working-tree language and test scans from crossing nested Git boundaries.
- 2026-09-21: Fixed the source installer to build from the cloned module with `go -C` and added an unrelated-working-directory integration test.
- 2026-09-21: Published `v0.1.0`, pinned the formula to the tagged archive and verified SHA256, and updated `marlonjd/homebrew-tap` at commit `703999a`.
- 2026-09-21: Verified the installed Homebrew binary and the published source installer against the four-repository Avia workspace.
- 2026-09-21: Browser verification passed for the workspace atlas and repository inventory with no console warnings, errors, or framework overlay.
- 2026-09-21: Final native gates passed: `go test ./...`, `go vet ./...`, `ginsights doctor .`, static report build, shell syntax, Ruby syntax, strict Homebrew audit, and `brew test marlonjd/tap/ginsights`.

## Next actions

None.

## Completion note

`ginsights serve . --workspace` now analyzes independent nested repositories without flattening their boundaries. Both public installation paths are operational: Homebrew installs stable `0.1.0`, and the source installer builds correctly from an unrelated Git working directory.

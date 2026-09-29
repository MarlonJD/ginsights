# 0009 — Live serve refresh

## Status

Implementation verified locally; `v0.1.1` published; Homebrew and running-server updates in progress.

## Goal

Keep served repository and workspace dashboards current without restarting the server, then publish the fix through the source installer and Homebrew.

## Scope

- Reload snapshots through the existing analysis and incremental Git cache paths.
- Check for updates while the dashboard is open, with a configurable `--refresh` duration defaulting to five seconds.
- Re-discover workspace repositories on refresh.
- Reload the browser only when report content changes and report refresh failures visibly.
- Publish `v0.1.1`, update the immutable Homebrew archive checksum and tap, and verify installed distribution paths.
- Apply the requested minimal harness adoption by consolidating existing instructions and removing speculative engineering rules.

## Non-goals

- No filesystem watcher dependency, frontend build, remote repository fetch, or implicit GitHub connector.
- Commit statistics continue to describe committed Git history; working-tree language and health signals retain their existing meaning.

## Acceptance criteria

- New commits, nested repository additions/removals, and existing filesystem metrics appear in HTML and JSON without restarting `serve`.
- Unchanged snapshots do not trigger repeated browser reloads due to generated timestamps.
- Concurrent refresh requests serialize analysis and preserve coherent rendered responses.
- A failed refresh retains the last successful report, signals failure, and retries on a later interval.
- Static reports remain self-contained and contain no live polling.
- Existing filters, cache options, workspace boundaries, and offline behavior remain intact.
- Native gates and isolated browser verification pass before publication.
- Harness guidance routes to the existing native gates and rejects speculative abstractions without adding a governance layer.
- The published Homebrew formula and installed binary use the verified release source.

## Verification

```bash
GOCACHE=/private/tmp/ginsights-go-build go test ./...
GOCACHE=/private/tmp/ginsights-go-build go test -race ./internal/server ./internal/app
GOCACHE=/private/tmp/ginsights-go-build go vet ./...
GOCACHE=/private/tmp/ginsights-go-build go run ./cmd/ginsights doctor .
GOCACHE=/private/tmp/ginsights-go-build go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

Browser verification uses an isolated temporary Git workspace and Chromium profile. Distribution verification includes the published archive SHA256, formula audit/test, Homebrew upgrade, and the published source installer.

Observed locally:

- `go test ./...`, targeted race tests, `go vet ./...`, doctor, static build, formatting, shell syntax, and Ruby syntax passed.
- The installed `0.1.0` reproduced stale HTML and JSON after a browser reload despite a new nested commit.
- The candidate browser flow passed for nested commits, repository addition/removal, uncommitted language changes, unchanged-content stability, and recovery from an intentional refresh failure. No JavaScript page errors were observed.
- The harness helper's adaptive `check` passed with zero errors and warnings after consolidating guidance and repairing Markdown routes.
- The final candidate passed the same browser flow with the default five-second interval, with zero errors during healthy refresh and only the expected HTTP 503 during the injected failure.
- Published source/tag `e353330` as `v0.1.1`. Downloaded its GitHub archive and verified SHA256 `656e280a0837c574e5066451e32410ebbf854c868892f311077aa95a7ff24791` before updating the formula.

## Decision log

- 2026-09-29: The current server renders one immutable startup snapshot, so a browser reload cannot refresh repository data.
- 2026-09-29: Use request-driven refresh with a minimum analysis interval and lightweight browser polling. This avoids scanning idle workspaces and reuses the existing incremental cache.
- 2026-09-29: The user authorized publication and updating all installation paths, including Homebrew, after verification.
- 2026-09-29: Baseline native tests passed. The default Go build cache is outside sandbox write roots; use a temporary `GOCACHE` for local verification.
- 2026-09-29: The user requested anti-overengineering harness adoption. Retain `docs/QUALITY_SCORE.md` because the native doctor consumes it, consolidate `CLAUDE.md`, fix instruction routes, and remove speculative future seams and checker backlogs.

## Next actions

1. Publish the verified Homebrew formula and upgrade the local installation.
2. Verify Homebrew and the published source installer.
3. Restart the existing Avia workspace server on its original loopback port with the upgraded binary.

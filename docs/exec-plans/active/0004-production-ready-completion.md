# 0004 — Production-ready active-plan completion

## Status

Active.

## Goal

Finish the active roadmap through 0001, 0002, and 0003 until the project is production-ready under the local/offline-first product contract.

## Scope

- Complete 0001 by adding and documenting `--since YYYY-MM-DD` for `serve`, `build`, and `json`.
- Complete 0002 by adding an optional disposable incremental analysis cache under `.ginsights-cache/`.
- Complete 0003 by adding an explicit optional GitHub API connector path with provenance labels and safe token handling.
- Keep the core product working offline without Node, Vite, frontend frameworks, remote assets, databases, daemons, or required network access.
- Keep detailed step, decision, verification, and commit notes in this file while work proceeds.

## Non-goals

- No branch changes unless explicitly requested.
- No required GitHub token or network dependency for core commands.
- No cache correctness shortcuts that can hide Git history changes.
- No GitHub Traffic inference from local Git data.

## Acceptance criteria

- 0001 is marked complete and its plan records the `--since` behavior.
- 0002 is marked complete with tests for cache hit, miss, stale, bypass, and clear behavior.
- 0003 is marked complete with tests for offline default behavior, explicit connector success/failure, provenance, and token redaction.
- Completed plans are moved to `docs/exec-plans/completed/` with completion notes.
- `go test ./...`, `go run ./cmd/ginsights doctor .`, and `go run ./cmd/ginsights build . --out /tmp/ginsights-report` pass at the end.

## Verification

```bash
GOCACHE=/tmp/ginsights-go-cache go test ./...
GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .
GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

## Decision log

- 2026-07-01: Treat this file as the durable progress ledger so detailed steps survive long-running work and context transitions.
- 2026-07-01: Complete work in milestones: 0001 first, then 0002, then 0003. Each milestone gets TDD tests, docs/plan updates, verification, and a small commit before moving on.

## Progress log

- 2026-07-01: Read `AGENTS.md`, `ARCHITECTURE.md`, 0001, 0002, 0003, `internal/app`, `internal/gitlog`, `internal/analyze`, and relevant tests before editing.
- 2026-07-01: Baseline before `--since`: `GOCACHE=/tmp/ginsights-go-cache go test ./...` passed; `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .` passed.
- 2026-07-01: Added failing TDD tests for JSON/build `--since` filtering, invalid date errors, and inclusive boundary filtering. Initial red result: `internal/app` failed to build because `filterCommitsSince` did not exist.
- 2026-07-01: Implemented `--since YYYY-MM-DD` parsing for `serve`, `build`, and `json`; filtering happens before snapshot aggregation.
- 2026-07-01: Documented `--since` in `README.md` and updated 0001 decision log/status.
- 2026-07-01: Verified 0001 with `GOCACHE=/tmp/ginsights-go-cache go test ./... -count=1`, `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .`, and `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights build . --out /tmp/ginsights-report`.
- 2026-07-01: Moved 0001 to `docs/exec-plans/completed/0001-mvp-github-style-dashboard.md` with a completion note.
- 2026-07-01: Committed 0001 milestone with message `feat(app): add since filtering`.
- 2026-07-01: Started 0002 incremental cache after fresh baseline: `GOCACHE=/tmp/ginsights-go-cache go test ./...` passed and `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .` passed.
- 2026-07-01: Added failing TDD tests for `internal/cache` hit/miss/stale/clear behavior. Initial red result: `Store`, `Collect`, `DefaultDir`, and `Clear` did not exist.
- 2026-07-01: Implemented `internal/cache` with disposable `commits-v1.json` storage, source hash reconciliation, stale prune on save, and clear support.
- 2026-07-01: Added failing TDD tests for `gitlog.Collector.ListHashes` and `CollectHashes`; implemented hash listing and selected-hash collection with existing log parsing.
- 2026-07-01: Added failing app integration tests for default cache creation, `--no-cache`, and `cache-clear`; implemented default cache integration for `serve`, `build`, and `json`.
- 2026-07-01: Documented cache behavior in `README.md`, `ARCHITECTURE.md`, `docs/product-specs/index.md`, and `docs/product-specs/cache-format.md`.
- 2026-07-01: Added parser/analyzer benchmark fixtures and verified them with `GOCACHE=/tmp/ginsights-go-cache go test ./internal/gitlog ./internal/analyze -run '^$' -bench Benchmark -benchtime=1x`.
- 2026-07-01: Moved 0002 to `docs/exec-plans/completed/0002-incremental-cache.md` with a completion note.
- 2026-07-01: Verified 0002 with `GOCACHE=/tmp/ginsights-go-cache go test ./... -count=1`, `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .`, and `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights build . --out /tmp/ginsights-report`.

## Next actions

1. Commit the completed 0002 milestone.
2. Start 0003 with connector boundary docs and TDD tests for safe optional GitHub behavior.

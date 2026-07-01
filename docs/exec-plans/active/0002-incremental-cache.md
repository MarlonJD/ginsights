# 0002 — Incremental analysis cache

## Status

Active.

## Goal

Add an optional local cache keyed by commit hash so repeated `serve` and `build` runs are faster on large repositories.

## Scope

- Cache raw parsed commits or aggregated per-commit stats.
- Store under `.ginsights-cache/` by default.
- Add `--no-cache` and `cache-clear` command only when the cache exists.
- Keep cache format documented and disposable.

## Non-goals

- No remote cache.
- No daemon.
- No cache correctness shortcuts that hide Git changes.

## Acceptance criteria

- First run populates cache.
- Second run avoids reparsing unchanged commits.
- Cache can be deleted with no data loss.
- Tests cover stale/missing cache behavior.

## Verification

```bash
go test ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

## Decision log

- 2026-07-01: Cache should be an optimization only; correctness source remains Git.

## Next actions

1. Add benchmark fixture for parser/analyzer.
2. Design cache structs in `internal/cache`.
3. Add cache invalidation tests.

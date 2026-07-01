# 0002 — Incremental analysis cache

## Status

Complete.

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
- 2026-07-01: Cache stores parsed commit records in `.ginsights-cache/commits-v1.json`, keyed by immutable commit hash. `git log --all --format=%H` remains the reachability source; stale cache entries are pruned on rewrite.
- 2026-07-01: Added `--no-cache` to `serve`, `build`, and `json`, plus `cache-clear [repo]` for disposable cleanup.

## Next actions

1. Continue with active plan 0003 optional GitHub connector.

## Completion note

- 2026-07-01: Completed optional incremental analysis cache with cache hit, missing, stale prune, bypass, and clear tests. Documented cache format in `docs/product-specs/cache-format.md` and verified parser/analyzer benchmark fixtures.

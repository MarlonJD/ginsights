# 0001 — MVP GitHub-style dashboard

## Status

Complete.

## Goal

Improve the current local report so `ginsights serve .` feels immediately close to a GitHub Insights page while remaining local/offline.

## Scope

- Better visual separation for Pulse, Contributors, Commits, Code frequency, Files, Languages, and Health.
- Branch/date filter design notes, even if not fully interactive yet.
- More informative empty states.
- Golden HTML fixture for renderer stability.

## Non-goals

- No GitHub token integration.
- No frontend framework.
- No database or long-running cache.

## Acceptance criteria

- `go run ./cmd/ginsights serve . --port 43117` renders the repo dashboard.
- `go run ./cmd/ginsights build . --out /tmp/ginsights-report` writes `index.html` and `data.json`.
- The report contains visible sections for all MVP tabs.
- Renderer tests cover at least one non-empty snapshot.

## Verification

```bash
go test ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

## Decision log

- 2026-07-01: Keep report self-contained; no Node/Vite until interaction complexity justifies it.
- 2026-07-01: Added a non-empty renderer golden fixture to catch unintended HTML output changes.
- 2026-07-01: Improved code-frequency and commit activity density inside the static renderer: heatmap days now fill complete weeks with `l0`-`l4` intensity classes, a compact legend, and per-day tooltips; code frequency now shows net change, a dense additions/deletions bar, and a weekly summary.
- 2026-07-01: Added `--since YYYY-MM-DD` to `serve`, `build`, and `json`. The flag filters Git-history metrics on or after the local start of the supplied day; working-tree signals such as languages and repository health remain current-checkout metrics.

## Next actions

1. Continue with active plan 0002 incremental cache.

## Completion note

- 2026-07-01: Completed MVP dashboard visual density and `--since YYYY-MM-DD` CLI filtering. Verified with `GOCACHE=/tmp/ginsights-go-cache go test ./... -count=1`, `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .`, and `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights build . --out /tmp/ginsights-report`.

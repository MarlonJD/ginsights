# 0001 — MVP GitHub-style dashboard

## Status

Active.

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

## Next actions

1. Add a renderer golden test fixture.
2. Improve code-frequency and heatmap visual density.
3. Add CLI flag for `--since` and document behavior.

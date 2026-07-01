# 0003 — Optional GitHub connector

## Status

Complete.

## Goal

Design an optional connector for GitHub server-side data such as stars, issues, PRs, and traffic where the user has token access.

## Scope

- Environment-based token discovery.
- Clear metric provenance labels: `local_git` vs `github_api`.
- Optional JSON merge into the report snapshot.
- No token persistence.

## Non-goals

- No bypassing GitHub plan/permission restrictions.
- No required network dependency for core commands.
- No background job in MVP.

## Acceptance criteria

- Core `serve`, `build`, and `json` still work offline.
- Connector failures degrade gracefully.
- Report labels GitHub API-sourced data explicitly.

## Verification

```bash
go test ./...
go run ./cmd/ginsights doctor .
```

## Decision log

- 2026-07-01: Keep connector behind explicit flag such as `--github-api`.
- 2026-07-01: Implemented `--github-api owner/name` for `serve`, `build`, and `json`; token discovery uses `GINSIGHTS_GITHUB_TOKEN` then `GITHUB_TOKEN`.
- 2026-07-01: Missing token and API failures degrade into `github.error` or `github.warnings` in the snapshot instead of failing local analysis.
- 2026-07-01: GitHub API-sourced report data is labeled with `github_api` provenance.

## Next actions

1. Continue production hardening through 0004 rollup.

## Completion note

- 2026-07-01: Completed explicit optional GitHub API connector with environment token discovery, token redaction tests, fake HTTP client tests, graceful missing-token behavior, JSON snapshot merge, report rendering, provenance labels, and connector documentation.

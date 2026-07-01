# 0003 — Optional GitHub connector

## Status

Active, not started.

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

## Next actions

1. Write connector product boundary doc.
2. Add interfaces and fake API fixture.
3. Add token redaction tests.

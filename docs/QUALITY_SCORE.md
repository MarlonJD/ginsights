# Quality score

## Required gates

```bash
go test ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

## Quality dimensions

- Correctness: Git parsing and aggregation match fixtures.
- Performance: medium repositories should render without surprising delays.
- Simplicity: new abstractions must serve at least two real call sites or a documented future seam.
- Agent readability: errors and docs should help Codex repair problems.
- Offline behavior: default commands should not require network access.

## Mechanical rules to add next

- package dependency checker;
- file size warning;
- report snapshot/golden test;
- command help text test;
- generated example report artifact in CI.

# Contributing

This project is designed for agent-assisted development.

## Local checks

```bash
go test ./...
go vet ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

## Pull request expectations

- Describe the user-visible change.
- Link or update the relevant execution plan for multi-step work.
- Include test updates for behavior changes.
- Keep unrelated cleanup out of the PR.

# Quality gates

## Required gates

```bash
go test ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

## Quality dimensions

- Correctness: Git parsing and aggregation match fixtures.
- Performance: medium repositories should render without surprising delays.
- Simplicity: direct code and existing dependencies meet the current requirement. An abstraction must reduce observed duplication or complexity; a hypothetical future seam does not justify it.
- Agent readability: errors and docs should help Codex repair problems.
- Offline behavior: default commands should not require network access.

## Verification scope

Use focused tests for changed behavior, then the required native gates above. Run browser verification when a change affects the served UI, and race checks when it changes concurrent execution. Repeat checks after relevant code changes or failures.

Create a new checker or automation only when it catches a concrete recurring defect that existing commands do not catch. Keep release and installation checks scoped to requested distribution work.

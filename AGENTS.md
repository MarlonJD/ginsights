# AGENTS.md

Purpose: make Codex productive quickly without turning this file into a manual. Treat this as the map. Deeper project knowledge lives in `docs/`.

## Start here

1. Read this file.
2. Read `ARCHITECTURE.md` for package boundaries.
3. Read the relevant active plan in `docs/exec-plans/active/`.
4. Run the baseline checks before changing behavior:
   - `go test ./...`
   - `go run ./cmd/ginsights doctor .`

## Product goal

Build a fast single-binary Go tool that renders GitHub-style local repository insights:

```bash
ginsights serve .
ginsights build . --out report
ginsights json .
```

Core mode must work offline from local Git data. GitHub Traffic data is not local Git data; only add it behind an explicit optional connector.

## Behavioral contract for Codex

Adapted from the referenced LLM coding guidelines, but made repo-specific:

- Think before coding. State assumptions inside the plan or PR notes when requirements are ambiguous.
- Prefer the smallest correct implementation. No speculative framework, plugin, or config layer.
- Make surgical changes. Every changed line should trace to the task or to cleanup caused by that task.
- Define success criteria before implementation and loop until verified.
- Do not hide confusion. If blocked, write the missing fact/tool/doc as an explicit follow-up in the plan.
- Match nearby style. Run `gofmt` for Go changes.
- Do not refactor unrelated code. Mention unrelated debt in `docs/exec-plans/tech-debt-tracker.md` instead.
- Write or update tests for changed behavior.

## Harness rules

- The repository is the source of truth. If Codex needs to know something later, commit it as Markdown, tests, fixtures, schemas, or code.
- Keep this file short. Add durable knowledge to `docs/` and link to it.
- Plans are first-class artifacts. Complex tasks require an active plan with goal, scope, verification, and decision log.
- Mechanical constraints beat prose. When a rule matters repeatedly, add a test, lint, or doctor check.
- Agent-readable output matters. CLI errors should say what failed and how to fix it.

## Architecture boundaries

Allowed dependency direction:

```text
cmd -> internal/app -> internal/{gitlog,analyze,report,server,doclint}
internal/report -> internal/analyze
internal/server -> internal/analyze, internal/report
```

Package rules are described in `ARCHITECTURE.md`. Keep packages boring and explicit.

## Done definition

Before claiming done:

```bash
go test ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

If a check cannot run, record the exact command, error, and reason in the response or plan.

## Useful docs

- `docs/product-specs/index.md` — product boundary and MVP
- `docs/exec-plans/active/0001-mvp-github-style-dashboard.md` — next implementation steps
- `docs/DESIGN.md` — UI direction
- `docs/QUALITY_SCORE.md` — quality gates
- `docs/RELIABILITY.md` — performance and failure expectations
- `docs/SECURITY.md` — local-only and token handling rules

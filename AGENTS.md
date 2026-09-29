# AGENTS.md

Purpose: make Codex productive quickly without turning this file into a manual. Treat this as the map. Deeper project knowledge lives in `docs/`.

## Start here

1. Read this file.
2. Read [ARCHITECTURE.md](ARCHITECTURE.md) for package boundaries.
3. Follow [docs/PLANS.md](docs/PLANS.md) and read a matching plan in `docs/exec-plans/active/` when one exists.
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
- Build in working end-to-end increments. Add abstractions and dependencies only for a present requirement or measured problem; a possible future use is insufficient.
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
- Reuse native tests and doctor checks. Add a guardrail only for a concrete recurring defect, and remove obsolete paths instead of preserving compatibility scaffolding.
- Reuse existing docs and commands before adding harness artifacts. Extra checkers, evidence stores, coverage matrices, certification, and maintenance automation require a concrete requested need.
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

- [Product specs](docs/product-specs/index.md) — product boundary and MVP
- [Planning policy](docs/PLANS.md) — plan scope and lifecycle
- [Design](docs/DESIGN.md) — UI direction
- [Quality gates](docs/QUALITY_SCORE.md) — native verification and simplicity
- [Reliability](docs/RELIABILITY.md) — performance and failure expectations
- [Security](docs/SECURITY.md) — local-only and token handling rules
- [Harness guidance](docs/references/harness-engineering-notes.md) — minimal adoption and artifact ownership

# Harness engineering notes

These notes are the project-specific operating model for building this repo with Codex.

## Principles applied here

1. The repo is the memory. Product decisions, plans, quality rules, and architecture constraints must be committed.
2. `AGENTS.md` is a map, not an encyclopedia. Durable details belong in `docs/`.
3. Plans are first-class. Complex work should update an execution plan before and after implementation.
4. Agent-readable systems beat human-only prose. Add tests, lint, doctor checks, fixtures, or scripts for repeated rules.
5. Boring technology is a feature. Favor Go stdlib and stable Git CLI behavior until there is evidence a dependency is worth it.
6. Feedback should become leverage. Review comments and recurring defects should become docs, tests, or mechanical checks.

## Repo memory structure

```text
AGENTS.md
ARCHITECTURE.md
docs/
  design-docs/
  exec-plans/
    active/
    completed/
    tech-debt-tracker.md
  product-specs/
  references/
  DESIGN.md
  FRONTEND.md
  PLANS.md
  PRODUCT_SENSE.md
  QUALITY_SCORE.md
  RELIABILITY.md
  SECURITY.md
```

## Current harness checks

Run:

```bash
go run ./cmd/ginsights doctor .
```

The doctor command currently checks that required docs exist, `AGENTS.md` stays short, and active plans contain executable sections. Add more checks whenever a repo rule becomes repetitive.

# Harness engineering notes

This repository uses the minimal adoption path from [apply-harness-engineering](https://github.com/MarlonJD/harness-engineering-skill/blob/main/SKILL.md), reviewed against package `0.2.1`. [AGENTS.md](../../AGENTS.md) remains the instruction entry point; architecture, plans, and native commands remain the existing authorities.

## Principles applied here

1. Keep the single-binary, offline product working after each end-to-end increment.
2. Implement present requirements with Go stdlib and existing dependencies. Introduce a new layer only when an observed problem justifies its maintenance cost.
3. Put durable decisions in existing docs, code, or behavior tests. Reuse an authority before creating another artifact.
4. Keep `AGENTS.md` concise and `CLAUDE.md` as a route to it. Preserve unique project decisions when merging guidance.
5. Follow the existing [planning policy](../PLANS.md). Small fixes do not need a new process or plan schema.
6. Verify changed behavior and run [native quality gates](../QUALITY_SCORE.md). A checker must catch a concrete defect, not merely validate another evidence artifact.

## Artifact boundary

Before adding a harness file, name its current consumer and check whether Git, a test, an existing document, or a native command already supplies the information. A working route may need no changes. Use consolidation or deletion when that preserves the same behavior with less maintenance.

Certification, governed scaffolds, coverage matrices, AI runtime contracts, evaluation suites, evidence stores, CI expansion, and maintenance schedules are added only for an explicit request or a concrete applicable risk. The default completion proof is the native project gate with concise observations.

## Current harness checks

Run:

```bash
go run ./cmd/ginsights doctor .
```

The doctor checks required documents, the size of `AGENTS.md`, and the sections of active plans. Keep it as the native structural check. Use product tests for product behavior and report `verified locally`, release state, and any missing external proof separately.

# CLAUDE.md

This repo is optimized for Codex through `AGENTS.md`, but the same operating rules apply to Claude or any coding agent.

Do not duplicate a large instruction manual here. Read:

- `AGENTS.md`
- `ARCHITECTURE.md`
- `docs/exec-plans/active/`

Agent behavior summary:

- think before coding;
- keep changes small and directly tied to the task;
- prefer simple, boring Go over speculative abstractions;
- write/update tests for changed behavior;
- run `go test ./...` and `go run ./cmd/ginsights doctor .` before done;
- if a durable rule is learned, encode it in docs, tests, or doctor checks.

Source adaptation note: these rules are a project-specific paraphrase of the behavioral guidelines referenced in the original CLAUDE.md URL supplied during project creation, plus the repository-harness practices documented in `docs/references/harness-engineering-notes.md`.

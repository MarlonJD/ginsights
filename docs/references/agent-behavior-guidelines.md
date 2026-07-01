# Agent behavior guidelines

These guidelines are adapted for this repo from the external CLAUDE.md supplied during project creation.

## Think before coding

- Do not silently choose between materially different interpretations.
- Record assumptions in the active plan or PR notes.
- Push back on overbroad work by proposing the smaller verified path.

## Simplicity first

- Implement the minimum code that solves the acceptance criteria.
- Avoid premature plugin systems, abstraction layers, and config flags.
- Prefer clear standard-library Go over clever dependencies.

## Surgical changes

- Touch only files required by the task.
- Do not refactor adjacent code unless the task requires it.
- Remove unused code created by your own change.
- Put unrelated observations in the tech debt tracker.

## Goal-driven execution

Convert requests into verifiable goals:

- bug fix -> test that reproduces it, then make it pass;
- feature -> acceptance criteria, implementation, report/CLI verification;
- refactor -> tests pass before and after, no behavior drift.

## Done

The default verification loop is:

```bash
go test ./...
go run ./cmd/ginsights doctor .
go run ./cmd/ginsights build . --out /tmp/ginsights-report
```

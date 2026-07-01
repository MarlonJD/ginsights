# Architecture

`ginsights` is intentionally boring: one Go binary, no frontend build step, no external runtime service, and no required network access.

## Package map

```text
cmd/ginsights
  -> internal/app
      -> internal/gitlog
      -> internal/analyze
      -> internal/report
      -> internal/server
      -> internal/doclint
```

## Responsibilities

- `cmd/ginsights`: only process entrypoint.
- `internal/app`: CLI flags, command dispatch, user-facing orchestration.
- `internal/gitlog`: shell out to `git`, collect raw commit/file-change events.
- `internal/analyze`: turn raw events into stable snapshot structs.
- `internal/report`: render snapshot to self-contained HTML and JSON.
- `internal/server`: host a local website from a snapshot.
- `internal/doclint`: enforce repo harness/documentation invariants.

## Dependency rules

- `internal/gitlog` must not import report/server/app packages.
- `internal/analyze` must not shell out or render HTML.
- `internal/report` must not execute Git commands.
- `internal/server` may serve rendered report data but must not mutate the repo.
- User-facing errors should include the command or file that failed and a practical next step.

## Data model boundary

Raw Git data enters through `internal/gitlog.Commit`. Everything presented to the UI goes through `internal/analyze.Snapshot`. New UI features should first add or extend snapshot fields, then render them.

## Future architecture hooks

Keep these as explicit future work, not hidden abstractions:

- incremental cache keyed by commit hash;
- optional GitHub connector for server-side traffic data;
- richer chart rendering;
- generated screenshots/video verification harness.

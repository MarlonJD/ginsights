# Disposable Cache Format

`ginsights` may create `.ginsights-cache/commits-v1.json` inside the analyzed repository.

Workspace mode keeps one cache inside each successfully analyzed repository. It does not create a shared cross-repository history cache at the workspace root unless that root is itself a repository.

## Purpose

The cache stores parsed `gitlog.Commit` records keyed by immutable commit hash so repeated `serve`, `build`, and `json` runs can avoid reparsing unchanged commits.

## Contract

- The cache is optional and disposable.
- Git history remains the correctness source.
- Deleting `.ginsights-cache/` must not lose user data.
- `--no-cache` bypasses load and save for one run.
- `ginsights cache-clear [repo]` removes the cache directory.
- Unknown, corrupt, missing, or unsupported cache contents are treated as cache misses.

## Shape

```json
{
  "version": 1,
  "updated_at": "2026-07-01T12:00:00Z",
  "commits": []
}
```

`commits` contains the same commit records used by the JSON report. Stale commits that are no longer reachable from `git log --all` are pruned when the cache is rewritten.

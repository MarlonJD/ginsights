# Multi-repository workspace analysis

## User contract

`ginsights serve . --workspace` treats the requested directory as a workspace containing one or more independent Git repositories. Without `--workspace`, existing single-repository behavior remains unchanged.

The same flag is available for static and JSON output:

```bash
ginsights build /path/to/workspace --workspace --out report
ginsights json /path/to/workspace --workspace
```

## Discovery

Workspace discovery scans for `.git` directory and file markers at the requested root and below it. It does not depend on the parent repository's tracked files, `.gitmodules`, or ignore rules, because a workspace may intentionally ignore nested repositories. Symlinked directory trees are not followed. Git metadata, disposable caches, `.state` runtime/release copies, vendored dependencies, and `node_modules` are pruned from discovery. `.state` trees also do not contribute language or test-presence signals to their parent repository. A repository inside `.state` can still be analyzed by passing that repository as the explicit command root.

## Analysis and aggregation

Each discovered Git root is analyzed independently. The workspace snapshot contains:

- aggregate commit, author, file-change, code-frequency, language, and recent-activity metrics;
- repository labels on aggregate commits and hot files;
- the complete per-repository snapshots and relative paths;
- repository-level health summaries;
- explicit per-repository failures when one repository cannot be analyzed.

The root repository's working-tree language and test scans stop at nested Git boundaries so nested files are not counted twice.

## Live serving

`serve --workspace` re-discovers Git roots and refreshes their independent snapshots while the dashboard is open. The minimum refresh interval defaults to five seconds and can be changed with `--refresh 30s`. HTML and JSON are updated together, using the existing incremental Git cache unless `--no-cache` is requested. Added and removed repositories appear on the next refresh without restarting the server.

The browser checks a content ETag and reloads only when report content changes. Generation timestamps alone do not cause reloads. A failed refresh preserves the last successful report, displays a retry status, and retries after the refresh interval. Manually reloading the page also refreshes data once that interval has elapsed. An idle server does not repeatedly scan the workspace.

Commit-based metrics require local Git history changes. Language and health signals retain their existing filesystem-based behavior. Static `build` and one-shot `json` output remain snapshots.

## Offline boundary

Workspace mode remains local-only. It does not infer GitHub repositories from remotes or contact GitHub. `--workspace` and the single-repository `--github-api owner/name` connector cannot be combined; analyze that repository separately when remote metrics are required.

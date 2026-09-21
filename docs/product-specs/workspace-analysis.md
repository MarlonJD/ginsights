# Multi-repository workspace analysis

## User contract

`ginsights serve . --workspace` treats the requested directory as a workspace containing one or more independent Git repositories. Without `--workspace`, existing single-repository behavior remains unchanged.

The same flag is available for static and JSON output:

```bash
ginsights build /path/to/workspace --workspace --out report
ginsights json /path/to/workspace --workspace
```

## Discovery

Workspace discovery scans for `.git` directory and file markers at the requested root and below it. It does not depend on the parent repository's tracked files, `.gitmodules`, or ignore rules, because a workspace may intentionally ignore nested repositories. Symlinked directory trees are not followed. Git metadata, disposable caches, vendored dependencies, and `node_modules` are pruned from discovery.

## Analysis and aggregation

Each discovered Git root is analyzed independently. The workspace snapshot contains:

- aggregate commit, author, file-change, code-frequency, language, and recent-activity metrics;
- repository labels on aggregate commits and hot files;
- the complete per-repository snapshots and relative paths;
- repository-level health summaries;
- explicit per-repository failures when one repository cannot be analyzed.

The root repository's working-tree language and test scans stop at nested Git boundaries so nested files are not counted twice.

## Offline boundary

Workspace mode remains local-only. It does not infer GitHub repositories from remotes or contact GitHub. `--workspace` and the single-repository `--github-api owner/name` connector cannot be combined; analyze that repository separately when remote metrics are required.

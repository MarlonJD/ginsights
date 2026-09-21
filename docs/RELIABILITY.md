# Reliability

## Expected behavior

- Missing Git binary should produce a clear error.
- Non-Git directories should produce a clear error.
- Binary file changes in `git numstat` should not crash analysis.
- Empty repositories should render a useful empty-state report.
- Workspace mode should preserve successful repository results and report a path-scoped error when another discovered repository fails.
- Static reports should be self-contained.

## Performance targets

Initial MVP target:

- small repo: under 1 second;
- medium repo: acceptable interactive wait;
- large repo: correctness first, then incremental cache.

Do not over-optimize before adding fixtures and benchmark coverage.

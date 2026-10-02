# 0010 — Workspace statistics and distribution verification

## Status

Implementation verified locally; publication and installed-binary verification are pending.

## Goal

Make the installed Avia workspace dashboard report current commit and line statistics, verify incremental updates, and publish the verified release through Homebrew.

## Scope

- Exclude disposable `.state` trees from workspace discovery, language detection, and test-presence detection.
- Cover cached multi-commit workspace line totals and coherent served HTML/JSON totals with regression tests.
- Verify the candidate against the four independent Avia repositories and raw Git numstat totals.
- Publish the source commit/tag and immutable Homebrew formula; install and verify the distributed binary.
- Resolve the obsolete source-installed executable that currently shadows the Homebrew command.

## Non-goals

- No new refresh framework: main already includes the v0.1.1 live refresh implementation.
- No changes to Avia application code or commits in its repositories.
- No remote repository fetching during analysis, branch creation, or cloud deployment.

## Acceptance criteria

- `.state` release copies do not contribute repositories, languages, or test-presence signals to their parent; explicitly analyzing a repository inside `.state` remains supported.
- Ignored independent component repositories remain discoverable.
- Cached, uncached, served, and raw Git line totals agree, including more than one new commit between analyses.
- HTML and JSON publish the same commit/addition/deletion/net totals after refresh.
- Avia workspace analysis contains exactly its four intended Git roots.
- Homebrew installs the verified immutable source and the default command resolves to that installation.

## Verification

- Baseline: `go test ./...` and `go run ./cmd/ginsights doctor .` passed.
- Reproduce the disposable repository discovery failure with a regression before fixing it.
- Run focused regressions, `go test ./...`, `go vet ./...`, doctor, and a static report build.
- Run an isolated installed-binary live-refresh scenario and compare Avia totals to direct Git output.
- Validate the published archive checksum, Ruby syntax, strict formula audit, install, and formula test.

## Decision log

- 2026-10-02: Installed `~/.local/bin/ginsights` is revision 185d46e; current main 672a8fd already has workspace support and live refresh. The old executable freezes every metric, so it alone does not explain the reported partial update.
- 2026-10-02: Ordinary incremental one- and two-commit fixtures correctly update line totals. No cache invalidation change is justified without a reproduced cache defect.
- 2026-10-02: Actual Avia analysis traverses nested `.state/release-workspaces/.../.state/releases/...` copies. Exclude the concrete disposable-state directory instead of introducing broad ignore rules or a configuration framework.
- 2026-10-02: The user authorized clone, implementation, verification, commit, push, Homebrew update, and local Avia verification.

## Next actions

Publish the verified source as v0.1.2, update the immutable formula and tap, install, resolve the obsolete command, and verify live refresh and the Avia dashboard.

## Local verification results

- The new discovery and language regressions failed before the implementation: disposable copies were discovered and their source bytes entered parent language totals.
- Focused regressions, the full Go test suite, `go vet ./...`, doctor, static HTML/JSON generation, and `git diff --check` passed after the final source change.
- Independent review found a test-presence walk that continued after success; returning `filepath.SkipAll` now stops it immediately.
- The candidate Avia snapshot contains exactly `.`, `apps/surveil`, `shared/auth`, and `shared/data`, with no analysis errors. Every repository's commit, addition, deletion, and net totals matched direct `git log --all --numstat` output. Observed aggregate: 1,195 commits, 1,464,105 additions, 692,604 deletions, net 771,501.
- The old-source freeze is verified; a current-source partial line-statistics freeze was not reproduced. The release does not claim a cache defect or invalidate correct cached history.

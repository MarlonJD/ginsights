# 0010 — Workspace statistics and distribution verification

## Status

Complete. v0.1.2 is published, installed through Homebrew, and verified in a live browser and the Avia workspace.

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

None. The verified Avia dashboard is running locally on port 43117 with a 30-second refresh interval and `--no-cache`, leaving component working trees free of new cache files.

## Local verification results

- The new discovery and language regressions failed before the implementation: disposable copies were discovered and their source bytes entered parent language totals.
- Focused regressions, the full Go test suite, `go vet ./...`, doctor, static HTML/JSON generation, and `git diff --check` passed after the final source change.
- Independent review found a test-presence walk that continued after success; returning `filepath.SkipAll` now stops it immediately.
- The candidate Avia snapshot contains exactly `.`, `apps/surveil`, `shared/auth`, and `shared/data`, with no analysis errors. Every repository's commit, addition, deletion, and net totals matched direct `git log --all --numstat` output. Observed aggregate: 1,195 commits, 1,464,105 additions, 692,604 deletions, net 771,501.
- The old-source freeze is verified; a current-source partial line-statistics freeze was not reproduced. The release does not claim a cache defect or invalidate correct cached history.

## Distribution and installed verification

- Published source commit `be06251c7cbaaff5be3d017df7d6734eab4b7d5a` as `v0.1.2`. The downloaded archive SHA256 is `26b346d900389d21f72a568f4b972805bbb99508f194235f4d159ea80f0ba43a`.
- Published source formula metadata in `190d52c` and the Homebrew tap update in `448cb72`. Ruby syntax and `brew audit --strict --formula marlonjd/tap/ginsights` passed before publishing the formula.
- `brew install marlonjd/tap/ginsights` and `brew test marlonjd/tap/ginsights` passed; `brew list --versions ginsights` reports `0.1.2`.
- Moved the old source-installed executable into a temporary recovery backup. The default command now resolves to `/opt/homebrew/bin/ginsights`, so later Homebrew upgrades affect the command actually used.
- The installed binary's live browser scenario advanced from 2 commits, +3/-0, net +3 to 4 commits, +6/-2, net +4 after two nested commits, without manual reload. Browser warnings/errors were empty; served JSON, cached data, and fresh uncached analysis agreed.
- The installed binary's Avia snapshot again contained exactly four intended repositories and matched raw Git commits/additions/deletions/net totals for each. Its observed aggregate remained 1,195 commits, +1,464,105/-692,604, net +771,501.
- Closed the disposable browser tab and stopped its server. The intended Avia dashboard is the only task-owned server left running.

## Completion note

The obsolete local installation has been replaced by the current Homebrew command. Workspace statistics exclude disposable release copies, existing live refresh is verified through the installed distribution, and regression tests cover the reported line-total behavior. No unproven cache repair was added and no Avia source changes were made.

# 0005 — Installation and distribution

## Status

Complete.

## Goal

Make `ginsights` installable through a Homebrew tap and a shell installer while keeping README clear and moving detailed rationale/install notes into focused docs.

## Scope

- Add a source-build shell installer that works from `curl | bash` and supports local options.
- Add a Homebrew formula template suitable for `marlonjd/tap`.
- Update README with concise install commands and clear project positioning.
- Add detailed installation and differentiation docs.
- Verify scripts and docs locally.

## Non-goals

- Do not publish an external Homebrew tap from this workspace without explicit user follow-up.
- Do not add Node, Vite, remote frontend assets, a database, or a daemon.
- Do not require GitHub API access for local install or local use.

## Acceptance criteria

- README has explicit Homebrew and `.sh` install commands.
- `docs/INSTALL.md` explains Homebrew tap setup, shell installer options, requirements, and release/tap maintenance.
- `docs/WHY.md` clearly explains why the project exists and how it differs from similar tools.
- Installer script has help/dry-run behavior and validates prerequisites.
- Homebrew formula can be syntax-checked locally.

## Verification

```bash
GOCACHE=/tmp/ginsights-go-cache go test ./...
GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .
bash -n scripts/install.sh
./scripts/install.sh --help
./scripts/install.sh --dry-run
ruby -c packaging/homebrew/Formula/ginsights.rb
```

## Decision log

- 2026-07-01: Use a source-build installer and source-build Homebrew formula so installability does not depend on prebuilt release artifacts existing yet.
- 2026-07-01: Document `marlonjd/tap` as the intended tap name, with the formula stored in this repo for copying into `homebrew-tap`.
- 2026-07-01: Keep README concise: direct install commands, quick use, short positioning, and links to `docs/INSTALL.md` and `docs/WHY.md` for detailed guidance.

## Progress log

- 2026-07-01: Baseline passed with `GOCACHE=/tmp/ginsights-go-cache go test ./...`; `doctor` exited 0 with the expected warning that no active plans existed before this plan was opened.
- 2026-07-01: Added `scripts/install.sh`, a source-build installer with `--help`, `--install-dir`, `--ref`, `--repo`, and `--dry-run`.
- 2026-07-01: Added `packaging/homebrew/Formula/ginsights.rb`, a source-build formula suitable for `marlonjd/tap`.
- 2026-07-01: Added `docs/INSTALL.md` for Homebrew tap, shell installer, source install, and verification details.
- 2026-07-01: Added `docs/WHY.md` explaining why the project exists and how it differs from GitHub Insights, raw Git commands, gitstats-style tools, SaaS analytics, and productivity scoring tools.
- 2026-07-01: Rewrote README to keep only install commands, quick use, product boundary, and doc links.
- 2026-07-01: Verified installer/formula with `bash -n scripts/install.sh`, `./scripts/install.sh --help`, `./scripts/install.sh --dry-run`, and `ruby -c packaging/homebrew/Formula/ginsights.rb`.
- 2026-07-01: Final verification passed with `GOCACHE=/tmp/ginsights-go-cache go test ./... -count=1`, `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights doctor .`, `GOCACHE=/tmp/ginsights-go-cache go run ./cmd/ginsights build . --out /tmp/ginsights-report`, installer checks, and formula syntax check.
- 2026-07-01: Fixed the tap owner typo from `marlonj/tap` to `marlonjd/tap`, copied `Formula/ginsights.rb` into the local `marlonjd/homebrew-tap` checkout, committed it there as `ed56886`, and pushed the tap.
- 2026-07-01: Homebrew now finds the formula but may require `brew trust marlonjd/tap` before loading it from the non-core tap.

## Next actions

1. Run `brew trust marlonjd/tap` locally if Homebrew refuses to load the formula as an untrusted tap, then run `brew install ginsights`.

## Completion note

- 2026-07-01: Added Homebrew tap packaging, source-build shell installer, install docs, and project rationale docs. README now points to detailed docs instead of carrying all details inline.

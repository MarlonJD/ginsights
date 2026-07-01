# ginsights

GitHub-style repository insights, locally.

`ginsights` is a Go-first starter for a single-binary local dashboard that analyzes data available from a Git repository and renders a GitHub Insights-like website. It is intentionally scoped to data that can be derived locally: commits, authors, additions/deletions, file churn, language mix, and repository health signals.

## Why this repo exists

This starter is built for Codex-driven development. The repo contains:

- a working Go CLI with `serve`, `build`, `json`, and `doctor` commands;
- a local static HTML report renderer;
- a short `AGENTS.md` map rather than a giant instruction blob;
- versioned product specs, architecture notes, quality rules, and execution plans;
- mechanical harness checks that Codex can run before and after changes.

## Quick start

After extracting the zip, initialize Git if the folder is not already a repository:

```bash
./scripts/bootstrap-local-git.sh
```

Then run:

```bash
go test ./...
go run ./cmd/ginsights serve . --port 43117
```

Open:

```text
http://127.0.0.1:43117
```

Build a static report:

```bash
go run ./cmd/ginsights build . --out report
python3 -m http.server 8000 -d report
```

Export JSON:

```bash
go run ./cmd/ginsights json . > insights.json
```

Validate the agent harness:

```bash
go run ./cmd/ginsights doctor .
```

## Current MVP scope

Implemented now:

- Git log collection through the installed `git` binary;
- contributor stats;
- weekly commit and code-frequency stats;
- hot files by churn;
- simple language byte breakdown;
- repository health checklist;
- local server and static report export;
- Codex-oriented docs and plans.

Not implemented yet:

- incremental cache;
- branch/date filters in the UI;
- richer GitHub-like charts;
- optional GitHub token integration for server-side data such as views/clones;
- generated screenshots/video verification.

See [`docs/exec-plans/active`](docs/exec-plans/active) for Codex-ready implementation plans.

## Important product boundary

This tool cannot infer GitHub Traffic data from a local repo. Views, unique visitors, clones, referrers, and popular content are GitHub server-side analytics. Keep the default product local/offline; add token-based collection only as an explicit optional connector.

## Project layout

```text
cmd/ginsights/          CLI entry point
internal/app/           command parsing and orchestration
internal/gitlog/        Git history collection
internal/analyze/       snapshot aggregation
internal/report/        HTML + JSON report generation
internal/server/        local website server
internal/doclint/       harness/documentation checks
docs/                   product, architecture, plans, and quality docs
```

## Development loop

```bash
make test
make lint
make doctor
make serve
```

When using Codex, start with `AGENTS.md`, then open the active execution plan that matches the task.

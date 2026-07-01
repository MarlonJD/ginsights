# Why ginsights exists

GitHub Insights is useful, but it is tied to GitHub's hosted product and mixes two very different kinds of data:

- data that exists in local Git history, such as commits, authors, file churn, additions, and deletions;
- server-side GitHub analytics, such as views, clones, referrers, and popular content.

`ginsights` exists to make the first category fast, local, reproducible, and easy to share as a static artifact. It intentionally does not pretend local Git history can reveal GitHub Traffic metrics. When GitHub API data is requested, it is opt-in and labeled as `github_api`.

## What makes it different

### Compared with GitHub Insights

GitHub Insights requires the repository to live on GitHub and some metrics are only available through GitHub's server-side APIs. `ginsights` works from a local checkout and keeps the default path offline. The output can be served locally, exported as static HTML, or emitted as JSON.

### Compared with `git log`, `git shortlog`, and ad hoc scripts

Raw Git commands are powerful but not a product surface. `ginsights` turns local Git history into a consistent dashboard with contributors, code frequency, hot files, language mix, health signals, provenance labels, and machine-readable JSON.

### Compared with gitstats-style report generators

Many Git history report tools produce broad historical charts. `ginsights` is shaped around a GitHub-style repository insight workflow: fast local command, static/offline report, explicit metric provenance, and agent-readable behavior for automation.

### Compared with engineering productivity analytics

`ginsights` is not for ranking developers or scoring people. It shows repository activity and maintenance signals. The product boundary is intentionally boring: explain what changed in the repo, where churn is concentrated, who contributed, and which metrics came from which source.

### Compared with SaaS code intelligence tools

SaaS tools can provide deeper hosted analytics, issue/PR workflows, and organization-wide reporting. `ginsights` is for the smaller, local-first job: run one binary against a repo and get a useful report without uploading code or configuring infrastructure.

## Design principles

- Local/offline core by default.
- One Go binary.
- Static HTML and JSON output.
- No frontend build step.
- No database or daemon.
- No hidden GitHub token requirement.
- Clear provenance: `local_git`, `working_tree`, and optional `github_api`.

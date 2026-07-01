# Optional GitHub API Connector

Core `ginsights` commands are local/offline by default. The GitHub API connector only runs when the user passes `--github-api owner/name`.

## Token discovery

The connector reads tokens from environment variables in this order:

1. `GINSIGHTS_GITHUB_TOKEN`
2. `GITHUB_TOKEN`

Tokens must not be accepted through CLI flags, persisted to cache files, rendered into reports, or printed in errors.

## Data and provenance

GitHub API-sourced data is merged into the JSON snapshot under `github` and shown in the HTML report only when requested. It is labeled with `github_api` provenance.

Current metrics:

- repository stars;
- forks;
- open issues count from repository metadata;
- traffic views and clones when the token has access.

Traffic endpoint failures are recorded as warnings while the rest of the local report remains usable.

## Failure behavior

Missing tokens, invalid repository slugs, API permission failures, and traffic restrictions must not fail local analysis. The snapshot should include a `github.error` or `github.warnings` entry instead.

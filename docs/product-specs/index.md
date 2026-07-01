# Product spec index

## One-liner

GitHub-style local repository insights as a fast Go single binary.

## Core user

A developer with a private or public Git repo who wants activity/contributor/code-frequency insight without uploading repo data or paying for unavailable hosted analytics.

## MVP jobs

- Analyze local Git history.
- Host a local dashboard and print the URL.
- Export a static HTML report for CI artifacts or GitHub Pages.
- Show only data that can be derived from local repo contents.

## Explicit non-goals for core mode

- Do not promise GitHub Traffic analytics locally.
- Do not require a GitHub token.
- Do not require Node, Python, Docker, or a frontend build system.
- Do not rank engineers by “productivity”; show repo activity and maintenance signals.

## Success criteria

`ginsights serve .` should produce a useful dashboard for a medium repo without configuration.

`ginsights build . --out report` should produce static files that can be opened by a browser.

`ginsights doctor .` should guide Codex toward fixing missing repo-harness pieces.

See also:

- [Disposable cache format](cache-format.md)

# Security

## Default mode

Default mode is local-only and does not send repository data anywhere.

## Future GitHub connector rules

- Connector must be opt-in.
- Token must come from environment or OS credential storage, never flags that are likely to be shell-history leaked.
- Never write tokens to reports, logs, cache files, or panic output.
- Clearly separate local Git-derived metrics from GitHub server-side analytics.

## Report safety

Report rendering uses Go `html/template` escaping. Do not switch to raw string concatenation for user/repo data.

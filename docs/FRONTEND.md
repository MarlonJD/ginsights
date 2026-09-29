# Frontend

The current frontend is server-rendered HTML from `internal/report`. Static exports remain self-contained snapshots. `internal/server` adds a small same-origin live update script only to served dashboards; it checks the report ETag and reloads the page when content changes.

## Rules

- No Node dependency until a concrete interaction requires it.
- Avoid remote assets so reports work offline.
- Keep CSS classes stable enough for screenshot tests later.
- Prefer progressive enhancement: the report must remain readable without JavaScript.

## Chart strategy

MVP charts are CSS/SVG-like HTML bars. Later, consider a tiny vendored chart module only if static HTML becomes too limited.

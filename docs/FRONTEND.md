# Frontend

The current frontend is server-rendered static HTML from `internal/report`.

## Rules

- No Node dependency until a concrete interaction requires it.
- Avoid remote assets so reports work offline.
- Keep CSS classes stable enough for screenshot tests later.
- Prefer progressive enhancement: the report must remain readable without JavaScript.

## Chart strategy

MVP charts are CSS/SVG-like HTML bars. Later, consider a tiny vendored chart module only if static HTML becomes too limited.

# Receipt Tracker

Mobile-first PWA for logging receipt values. Part of the Cellarium monorepo.

## Directory Layout

```
receipt-tracker/
  main.go           # entry point, server setup, embeds frontend
  handler.go        # POST /api/entries handler
  handler_test.go   # handler tests (stdlib only)
  envload.go        # hierarchical .env loading
  envload_test.go   # env loading tests
  db/
    migrations/     # golang-migrate SQL files
    queries/        # sqlc query definitions
    sqlc/           # generated Go code (committed)
  frontend/         # vanilla JS PWA (no build step)
```

## Migrations

Add new migration files in `db/migrations/` following the naming convention:
```
NNNN_description.up.sql
NNNN_description.down.sql
```

Run with `migrate` CLI — the app binary does not run migrations.

## sqlc Workflow

1. Edit queries in `db/queries/`
2. Run `sqlc generate` from this directory
3. Commit the generated code in `db/sqlc/`

## Handler Pattern

- Handler functions take a `sqlc.Querier` interface and return `http.Handler`
- This enables testing with stub implementations (no real database needed)
- Value is transmitted as a string in JSON to avoid float precision issues

## Frontend

- Vanilla HTML/CSS/JS, no build step
- Embedded into the Go binary via `//go:embed`
- Uses IndexedDB for offline queue, service worker for caching

## Formatting

Run `gofmt -w .` before committing to ensure all Go code is properly formatted.

## Testing

- Standard library only (`testing`, `net/http/httptest`)
- Red/green TDD: write failing tests first, then implement
- Run: `go test ./...`

## Copyright

All source files must include the GNU GPLv3 copyright header. See root `CLAUDE.md` for the template.

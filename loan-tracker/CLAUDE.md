# Loan Tracker

Web app for tracking repayment of a single loan. Part of the Cellarium monorepo.

## Directory Layout

```
loan-tracker/
  main.go           # entry point, server setup, embeds templates + migrations
  handler.go        # GET /, POST /setup, POST /payment handlers
  handler_test.go   # handler tests (stdlib only)
  envload.go        # hierarchical .env loading
  envload_test.go   # env loading tests
  db/
    migrations/     # golang-migrate SQL files
    queries/        # sqlc query definitions
    sqlc/           # generated Go code (committed)
  frontend/         # static assets (CSS)
  templates/        # Go html/template files
```

## Data Model

Single table `loans.entries`:
- First row: loan amount (positive number)
- Subsequent rows: repayments (negative numbers)
- `SUM(amount)` = remaining balance

## Migrations

Add new migration files in `db/migrations/` following the naming convention:
```
NNNN_description.up.sql
NNNN_description.down.sql
```

Run with the app binary: `./loan-tracker -migrate up`

## sqlc Workflow

1. Edit queries in `db/queries/`
2. Run `sqlc generate` from this directory
3. Commit the generated code in `db/sqlc/`

## Handler Pattern

- Handler functions take a `sqlc.Querier` interface and return `http.Handler`
- This enables testing with stub implementations (no real database needed)
- All forms use POST with PRG (Post/Redirect/Get) pattern

## Frontend

- Server-side rendered HTML via Go `html/template`
- No JavaScript required for any functionality
- Static CSS embedded via `//go:embed`
- `Cache-Control: no-store` on dashboard to prevent caching financial data

## Formatting

Run `gofmt -w .` before committing to ensure all Go code is properly formatted.

## Testing

- Standard library only (`testing`, `net/http/httptest`)
- Red/green TDD: write failing tests first, then implement
- Run: `go test ./...`

## Copyright

All source files must include the GNU GPLv3 copyright header. See root `CLAUDE.md` for the template.

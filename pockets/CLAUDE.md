# Pockets

Virtual bank account tracker with monthly auto-top-ups and forecasting. Part of the Cellarium monorepo.

## Directory Layout

```
pockets/
  main.go               # entry point, server setup, embeds templates + migrations
  envload.go            # hierarchical .env loading
  envload_test.go       # env loading tests
  money.go              # numeric/float64 conversion, amount parsing/formatting
  money_test.go         # money helper tests
  colours.go            # 8 preset colour constants and validation
  colours_test.go       # colour validation tests
  handler_dashboard.go  # GET / — account cards with balances
  handler_account.go    # CRUD for accounts
  handler_transaction.go # CRUD for transactions
  handler_topup.go      # top-up rules management + auto-generation
  handler_forecast.go   # per-account and all-accounts forecast
  handler_test.go       # all handler tests (stdlib only)
  static/
    style.css           # WCAG 2.2 AAA monochrome stylesheet
    edit.js             # progressive enhancement for inline editing
  templates/            # Go html/template files with shared layout partial
  db/
    migrations/         # golang-migrate SQL files
    queries/            # sqlc query definitions
    sqlc/               # generated Go code (committed)
```

## Data Model

Three tables in `pockets` schema:
- `accounts` — virtual bank accounts with name, icon, colour, optional target
- `transactions` — inflows/outflows with amount always positive, is_inflow flag
- `topup_rules` — monthly auto-top-up amounts with effective dates

Balance = `SUM(CASE WHEN is_inflow THEN amount ELSE -amount END)`.

## Migrations

Run with: `./pockets -migrate up`

## sqlc Workflow

1. Edit queries in `db/queries/`
2. Run `sqlc generate` from this directory
3. Commit the generated code in `db/sqlc/`

## Handler Pattern

- Handler functions take a `sqlc.Querier` interface and `*template.Template`, return `http.Handler`
- All forms use POST with PRG (Post/Redirect/Get) pattern
- Templates use `{{template "layout" .}}` for shared layout

## Formatting

Run `gofmt -w .` before committing.

## Testing

- Standard library only (`testing`, `net/http/httptest`)
- Red/green TDD
- Run: `go test ./...`

## Copyright

All source files must include the GNU GPLv3 copyright header.

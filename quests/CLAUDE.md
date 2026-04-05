# Quests

Gamified todo list where quests fail if not completed by their date. Part of the Cellarium monorepo.

## Structure

- `quest.go` — core logic: `ensureFailedQuests`, `createNextRecurrence`
- `notify.go` — background ticker, push notification sending
- `handler_*.go` — HTTP handlers
- `db/migrations/` — PostgreSQL schema migrations
- `db/queries/` — sqlc query definitions
- `db/sqlc/` — generated code (committed)
- `templates/` — HTML templates (full pages + `_nav` partial)
- `static/` — CSS, JS assets

## Key behaviours

- Quest fails when visited the next day after its date (quest_date < today)
- Recurring quests: `every` type uses quest_date as base; `after_completion` uses completion time
- Push notifications require VAPID env vars: VAPID_PRIVATE_KEY, VAPID_PUBLIC_KEY, VAPID_SUBJECT
- Service runs on :8085 by default

## Commands

```
go test ./...     — run all tests
sqlc generate     — regenerate DB code after query/migration changes
gofmt -w .        — format all Go files
go build ./...    — build
./quests -migrate up   — run migrations
```

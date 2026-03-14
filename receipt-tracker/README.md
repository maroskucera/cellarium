# Receipt Tracker

A mobile-first PWA for quickly logging numeric values from paper receipts. Runs inside a Tailscale tailnet (no auth needed). Write-only — reading/querying is done via direct database access.

## Prerequisites

- Go 1.22+
- PostgreSQL
- [sqlc](https://sqlc.dev/) CLI (for regenerating query code)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (for running migrations)

## Setup

1. Create a PostgreSQL database:
   ```
   createdb cellarium
   ```

2. Run migrations:
   ```
   migrate -database "postgres://localhost:5432/cellarium?sslmode=disable" -path db/migrations up
   ```

3. Configure environment (create `.env.local` with your database URL):
   ```
   DATABASE_URL=postgres://user:pass@localhost:5432/cellarium?sslmode=disable
   ```

4. Run:
   ```
   go run .
   ```

## API

### POST /api/entries

Create a receipt entry.

**Request:**
```json
{
  "value": "42.50",
  "entry_date": "2026-03-14",
  "note": "groceries"
}
```

- `value` (required): Decimal amount as string
- `entry_date` (optional): Date in `YYYY-MM-DD` format, defaults to today
- `note` (optional): Free-text note

**Response (201 Created):**
```json
{
  "id": 1,
  "value": "42.50",
  "entry_date": "2026-03-14",
  "note": "groceries",
  "created_at": "2026-03-14T12:00:00Z"
}
```

## Build

```
go build -o receipt-tracker .
```

Produces a single static binary with the frontend embedded.

## Deployment

Copy the binary to the target machine and run it. Set `DATABASE_URL` and optionally `LISTEN_ADDR` (default `:8080`) via environment variables or `.env.local`.

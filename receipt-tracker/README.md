# Receipt Tracker

A mobile-first server-rendered form for quickly logging numeric values from paper receipts. Runs inside a Tailscale tailnet (no auth needed). Write-only — reading/querying is done via direct database access.

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

## Usage

Open `http://localhost:8080` in a browser. Fill in the amount (required), date (defaults to today), and an optional note, then submit. The form uses the Post-Redirect-Get pattern — after a successful submission you'll see a confirmation message.

## Build

```
go build -o receipt-tracker .
```

Produces a single static binary with templates and CSS embedded.

## Deployment

Copy the binary to the target machine and run it. Set `DATABASE_URL` and optionally `LISTEN_ADDR` (default `:8080`) via environment variables or `.env.local`.

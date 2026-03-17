# Loan Tracker

A simple web app for tracking repayment of a single loan. Part of the [Cellarium](../README.md) monorepo.

## Features

- Track a single loan with repayment history
- View repayment stats: amount remaining, percentage repaid, projected completion
- CSS-only progress bar with milestone markers
- No JavaScript required — works with JS disabled
- All financial data fetched fresh from database on every page load

## Setup

1. Set the `DATABASE_URL` environment variable (or use `.env` / `.env.local` files)
2. Run migrations: `./loan-tracker -migrate up`
3. Start the server: `./loan-tracker`

The app listens on `:8082` by default. Override with the `LISTEN_ADDR` environment variable.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string |
| `LISTEN_ADDR` | `:8082` | Address to listen on |

## License

GNU GPLv3 — see [COPYING](../COPYING) for details.

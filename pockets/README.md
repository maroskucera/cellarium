# Pockets

Virtual bank account tracker with monthly auto-top-ups and forecasting. Part of the [Cellarium](../) monorepo.

## Features

- Multiple virtual accounts with emoji icons and colour coding
- Manual inflow/outflow transactions
- Automatic monthly top-up generation from configurable rules
- Per-account and all-accounts balance forecasting (6/12 months)
- Target amounts with progress tracking
- WCAG 2.2 AAA monochrome design with light/dark mode

## Quick Start

```bash
# Set DATABASE_URL in .env.local
echo 'DATABASE_URL=postgres://user:pass@localhost:5432/cellarium' > .env.local

# Run migrations
go build -o pockets . && ./pockets -migrate up

# Start server
./pockets
# → http://localhost:8083
```

## License

GNU GPLv3 — see [COPYING](../COPYING).

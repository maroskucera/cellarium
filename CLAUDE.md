# Cellarium

Monorepo of small, single-purpose Go mini-apps sharing a central PostgreSQL database.

## Structure

Each app lives in its own directory with its own `go.mod`, `CLAUDE.md`, and `README.md`.

## Common Patterns

- **SQL**: sqlc for type-safe queries, golang-migrate for schema migrations
- **Dependencies**: pgx/v5, joho/godotenv
- **Frontend**: Vanilla JS, embedded via `//go:embed`, no build step
- **Binaries**: Static, single-binary deployment
- **Auth**: None — apps run inside a Tailscale tailnet
- **Endpoints**: Write-only; reading/querying via direct DB access

## Environment Variables

Hierarchy (highest priority first):
1. Real environment variables (never overwritten)
2. App-level `.env.local` (gitignored, for secrets)
3. App-level `.env` (committed, non-secret defaults)
4. Root-level `.env.local` (gitignored)
5. Root-level `.env` (committed, non-secret defaults)

Uses `joho/godotenv` which does not override already-set vars, so files are loaded in priority order (highest first).

## sqlc

Generated code is committed to git. Run `sqlc generate` from the app directory after changing queries or migrations.

## Formatting

All Go code must be formatted with `gofmt`. Run `gofmt -w .` from the app directory before committing.

## Testing

- Standard library only (`testing`, `net/http/httptest`) — no testify or other test frameworks
- Red/green TDD: write failing tests first, then implement

## Commits

Conventional Commits required: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`, etc.

## License

GNU GPLv3. All source files (Go, JS, CSS, HTML, SQL) must include the copyright header:

```
<one line description of the program>
Copyright (C) <year> <name of author>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```

Wrapped in appropriate comment syntax for each language.

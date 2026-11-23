# pglet

A fast, single-binary PostgreSQL database browser. Connect to any PostgreSQL server and explore schemas, run queries, view table data, and manage saved queries — all from a modern web UI.

## Features

- **Schema browser** — explore tables, views, materialized views, functions, sequences, and types across all schemas
- **SQL editor** — Monaco-based editor with syntax highlighting and multi-tab support
- **Table inspector** — browse rows with sorting and pagination, view columns, indexes, constraints, and size info
- **Query history** — automatic logging of every query with duration and row counts
- **Saved queries** — organize frequently used queries; share them through `.pglet/queries/*.sql` files in a project
- **Safe writes** — optionally hold any query that changes data in a transaction until you commit or roll back
- **Export** — download query results as CSV or JSON
- **AI SQL generation** — natural language to SQL via Claude API (optional, requires `ANTHROPIC_API_KEY`)
- **Server monitoring** — view active queries (`pg_stat_activity`), server settings, and table statistics
- **Multi-database** — switch between databases on the same server without reconnecting
- **Single binary** — frontend is embedded via `go:embed`, no separate web server needed

## Quick Start

```bash
# Build from source
make build

# Start and connect via the UI
./pglet

# Or connect on startup
./pglet --url postgres://user@localhost:5432/mydb

# Or with individual flags (the password can come from PGPASSWORD)
PGPASSWORD=secret ./pglet --host localhost --port 5432 --user postgres --db mydb
```

pglet prints a link such as `http://localhost:8081/?token=…`. Open it once (or pass `--open`) and the browser stays signed in; the token is kept in the data directory, so the same link works after a restart.

## Security

pglet holds a live connection to your database, so it guards the API:

- It listens on `localhost` only by default, and rejects requests addressed to any other host name, which defeats DNS-rebinding attacks from websites you visit.
- Every API request needs the sign-in cookie set by the token link, plus an `X-Pglet` header that other websites cannot send.
- Cross-origin calls are refused unless you allow an origin with `--cors-origin`.

`--bind 0.0.0.0` makes pglet reachable from your network; anyone with the link can then use your database, so only do this on a network you trust.

## Installation

### From source

Requires Go 1.26+ and Node.js 20+ with pnpm.

```bash
git clone https://github.com/macleodmac/pglet.git
cd pglet
make deps     # install frontend + Go dependencies
make build    # build production binary
```

## Usage

```
pglet [flags]

Connection:
  --url <url>       PostgreSQL connection URL
  --host <host>     Database host
  --port <port>     Database port (default: 5432)
  --user <user>     Database user
  --pass <pass>     Database password (prefer PGPASSWORD: flags show up in ps)
  --db <name>       Database name
  --ssl <mode>      SSL mode (default: disable)

Server:
  --bind <addr>     Bind address (default: localhost)
  --listen <port>   Listen port (default: 8081)
  --prefix <path>   URL prefix (default: /)
  --open            Open browser on start
  --store-dir <dir> Data directory (default: ~/.pglet/)
  --dev             Development mode (UI served by Vite, no sign-in token)
  --cors-origin <o> Allow API calls from this origin (repeatable)

Other:
  -h, --help        Show help
  -v, --version     Show version
```

## Development

```bash
make dev          # Run Vite dev server (:5173) + Go backend (:8081) concurrently
make generate     # Regenerate Go + TypeScript from openapi.yaml
make build        # Build production binary
make deps         # Install all dependencies
make clean        # Remove build artifacts
make db-up        # Start the dev Postgres (dev/docker-compose.yml, seeded from dev/seed.sql)
```

Component stories use [Ladle](https://ladle.dev): `cd frontend && pnpm ladle`.

Use `pnpm`, not npm.

### API-first workflow

The API is defined in `openapi.yaml` (OpenAPI 3.0.3). To add or modify endpoints:

1. Edit `openapi.yaml`
2. Run `make generate` to regenerate Go types/routing and TypeScript SDK
3. Implement the handler in `internal/api/handlers_<domain>.go`
4. Add business logic in `internal/service/<domain>.go`
5. Add the TanStack Query hook in `frontend/src/api/queries.ts`

## Architecture

```
openapi.yaml (source of truth)
  ├─► oapi-codegen  → internal/api/openapi.gen.go   (types + ServerInterface + routing)
  └─► hey-api       → frontend/src/api/generated/ (TS types + SDK)

Frontend (React 19, Vite 7, Tailwind v4, TanStack Query v5, Monaco, Zustand)
  → pnpm build → static/dist/
  → go:embed   → single Go binary
```

### Backend

| Layer | Directory | Purpose |
|-------|-----------|---------|
| HTTP | `internal/api/` | Thin handlers implementing `ServerInterface`. Parse request, call service, write response. |
| Service | `internal/service/` | All business logic. Single `Service` struct with methods namespaced by file. |
| Client | `internal/client/` | PostgreSQL wrapper around `lib/pq`. Schema introspection, query execution. |
| Repository | `internal/repository/` | bbolt-based persistence for saved queries, history, settings, tab state. |
| AI | `internal/ai/` | Anthropic Claude API client for SQL generation. |

Middleware chain (net/http), outermost first:
- Panic recovery
- CORS for `--cors-origin` origins only
- Guard: loopback `Host` check, sign-in cookie, `X-Pglet` header
- Structured logging with `log/slog`
- OpenAPI request validation via `oapi-codegen/nethttp-middleware`

### Frontend

| Directory | Purpose |
|-----------|---------|
| `frontend/src/api/` | Generated SDK + TanStack Query hooks (`queries.ts`) |
| `frontend/src/components/` | React components: `layout/`, `editor/`, `tabs/`, `results/`, `browser/`, `ai/`, `saved-queries/`, `ui/` |
| `frontend/src/stores/` | Zustand stores: `tabs.ts`, `connection.ts`, `theme.ts` |

## AI Features

Set the `ANTHROPIC_API_KEY` environment variable to enable AI features:

- **SQL generation** — describe what you want in natural language and get PostgreSQL SQL
- **Query suggestions** — contextual query ideas based on your database schema
- **Tab naming** — automatic short names for query tabs (falls back to heuristic without API key)

```bash
ANTHROPIC_API_KEY=sk-ant-... ./pglet --url postgres://localhost/mydb
```

## Running queries

- Results keep the first 10,000 rows; the toolbar says when a result was cut short, and **All rows** downloads the full result by re-running the query in a read-only transaction.
- CSV and JSON export save the rows on screen; nothing is re-run.
- Outside a transaction, each run gets a fresh session: settings changed with `SET` (and `SET ROLE`) are reset afterwards. Use **Begin** to keep a session, and its settings, across runs.
- `BEGIN`, `COMMIT` and `ROLLBACK` can't be typed while a transaction is open or safe writes is on; use the buttons. Explain and Analyze take a single statement, and Analyze is always rolled back.
- Connecting, reconnecting or switching database rolls back every open transaction.

## Data Storage

pglet keeps its private data in `~/.pglet/` (or `--store-dir <path>`):

- `pglet.db` — bbolt database: query history, tab state, saved connections, saved queries and the sign-in token

Shared queries are `.sql` files in the nearest `.pglet/queries/` directory at or above the working directory (or `~/.pglet/queries/`). The files are the source of truth: editing or deleting a shared query in pglet changes the file. Each file starts with a header:

```sql
-- pglet-query
-- @title: Active users
-- @description: Users seen in the last 30 days
-- @database: *
-- @tags: users

SELECT * FROM users WHERE last_seen > now() - interval '30 days';
```

## License

MIT

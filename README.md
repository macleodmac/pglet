# pglet

A fast, single-binary PostgreSQL database browser. Connect to any PostgreSQL server and explore schemas, run queries, view table data, and manage saved queries — all from a modern web UI.

## Features

- **Schema browser** — explore tables, views, materialized views, functions, sequences, and types across all schemas
- **SQL editor** — Monaco-based editor with syntax highlighting and multi-tab support
- **Table inspector** — browse rows with sorting and pagination, view columns, indexes, constraints, and size info
- **Query history** — automatic logging of every query with duration and row counts
- **Saved queries** — organize and share frequently used queries, with file-based import from `.pglet/queries/`
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
./pglet --url postgres://user:pass@localhost:5432/mydb

# Or with individual flags
./pglet --host localhost --port 5432 --user postgres --db mydb
```

Open `http://localhost:8081` in your browser.

## Installation

### From source

Requires Go 1.22+ and Node.js 18+ with pnpm.

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
  --pass <pass>     Database password
  --db <name>       Database name
  --ssl <mode>      SSL mode (default: disable)

Server:
  --bind <addr>     Bind address (default: localhost)
  --listen <port>   Listen port (default: 8081)
  --prefix <path>   URL prefix (default: /)
  --open            Open browser on start
  --store-dir <dir> Data directory (default: ~/.pglet/)
  --dev             Development mode (CORS, verbose logging)
  --cors            Enable CORS

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
```

Use `pnpm`, not npm.

### API-first workflow

The API is defined in `openapi.yaml` (OpenAPI 3.0.3). To add or modify endpoints:

1. Edit `openapi.yaml`
2. Run `make generate` to regenerate Go types/routing and TypeScript SDK
3. Implement the handler in `pkg/api/handlers_<domain>.go`
4. Add business logic in `pkg/service/<domain>.go`
5. Add the TanStack Query hook in `frontend/src/api/queries.ts`

## Architecture

```
openapi.yaml (source of truth)
  ├─► oapi-codegen  → pkg/api/openapi.gen.go   (types + ServerInterface + routing)
  └─► hey-api       → frontend/src/api/generated/ (TS types + SDK)

Frontend (React 19, Vite 7, Tailwind v4, TanStack Query v5, Monaco, Zustand)
  → pnpm build → frontend/dist/
  → go:embed   → single Go binary
```

### Backend

| Layer | Directory | Purpose |
|-------|-----------|---------|
| HTTP | `pkg/api/` | Thin handlers implementing `ServerInterface`. Parse request, call service, write response. |
| Service | `pkg/service/` | All business logic. Single `Service` struct with methods namespaced by file. |
| Client | `pkg/client/` | PostgreSQL wrapper around `lib/pq`. Schema introspection, query execution. |
| Repository | `pkg/repository/` | bbolt-based persistence for saved queries, history, settings, tab state. |
| AI | `pkg/ai/` | Anthropic Claude API client for SQL generation. |

Middleware chain (net/http):
- OpenAPI request validation via `oapi-codegen/nethttp-middleware`
- Structured logging with `log/slog`
- Panic recovery
- CORS (development mode)

### Frontend

| Directory | Purpose |
|-----------|---------|
| `frontend/src/api/` | Generated SDK + TanStack Query hooks (`queries.ts`) |
| `frontend/src/components/` | React components: `layout/`, `editor/`, `tabs/`, `results/`, `ai/`, `saved-queries/` |
| `frontend/src/stores/` | Zustand stores: `tabs.ts`, `connection.ts`, `theme.ts` |

## AI Features

Set the `ANTHROPIC_API_KEY` environment variable to enable AI features:

- **SQL generation** — describe what you want in natural language and get PostgreSQL SQL
- **Query suggestions** — contextual query ideas based on your database schema
- **Tab naming** — automatic short names for query tabs (falls back to heuristic without API key)

```bash
ANTHROPIC_API_KEY=sk-ant-... ./pglet --url postgres://localhost/mydb
```

## Data Storage

pglet stores its data in a `.pglet/` directory:

- **Default location**: `~/.pglet/`
- **Project-local**: if a `.pglet/` directory exists in or above the current working directory, it is used instead
- **Custom**: `--store-dir <path>`

Contents:
- `pglet.db` — bbolt database (saved queries, history, settings, tab state)
- `queries/` — shared query files (`.sql`) that are imported on startup

## License

MIT

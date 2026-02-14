# pglet

A single-binary PostgreSQL browser: Go backend (Gin + SQLite + lib/pq) with an embedded React SPA.

## Architecture

```
openapi.yaml (source of truth)
  ├─► oapi-codegen  → pkg/api/openapi.gen.go   (types + ServerInterface)
  └─► hey-api       → frontend/src/api/generated/ (TS types + SDK)

Frontend (React 19, Vite 6, Tailwind v4, TanStack Query v5, Monaco, Zustand)
  → pnpm build → frontend/dist/
  → go:embed   → single Go binary
```

## Project Layout

```
main.go                  # CLI flags, Gin setup, server start
openapi.yaml             # OpenAPI 3.0.3 spec
oapi-codegen.yaml        # codegen config
pkg/
  api/                   # HTTP handlers (implement ServerInterface)
    handlers_connection.go
    handlers_schema.go
    handlers_query.go
    handlers_export.go
    handlers_saved.go
    handlers_history.go
    handlers_ai.go
  client/                # PostgreSQL client wrapper (lib/pq)
  store/                 # SQLite persistence (saved queries, history, settings)
  ai/                    # Anthropic API integration
frontend/src/
  api/                   # Generated SDK + TanStack Query hooks (queries.ts)
  components/            # layout/, editor/, tabs/, results/, ai/, saved-queries/
  stores/                # Zustand (tabs.ts, connection.ts, theme.ts)
static/                  # go:embed entry point
```

## Dev Workflow

```bash
make dev          # Vite :5173 + Go :8081 concurrently
make generate     # Regen Go + TS from openapi.yaml
make build        # Production binary
```

Use `pnpm`, not npm.

## API Endpoints

| Group | Endpoints |
|-------|-----------|
| Connection | `POST /api/connect`, `POST /api/disconnect`, `POST /api/switchdb`, `GET /api/connection`, `GET /api/databases`, `GET /api/info` |
| Schema | `GET /api/schemas`, `GET /api/objects`, `GET /api/tables/{table}`, `GET /api/tables/{table}/rows\|info\|indexes\|constraints`, `GET /api/tables_stats`, `GET /api/server_settings`, `GET /api/activity` |
| Query | `POST /api/query`, `POST /api/explain`, `POST /api/analyze`, `POST /api/query/cancel` |
| Export | `GET /api/export` |
| Saved Queries | `GET\|POST /api/saved-queries`, `GET\|PUT\|DELETE /api/saved-queries/{id}` |
| History | `GET\|DELETE /api/history` |
| Settings | `GET\|PUT /api/settings` |
| AI | `POST /api/ai/generate` |

## Key Patterns

- **Spec-first**: always edit `openapi.yaml` first, then `make generate`
- **Handler convention**: `pkg/api/handlers_<domain>.go` implements `ServerInterface` methods
- **TanStack Query hooks**: `frontend/src/api/queries.ts` wraps generated SDK with `unwrap()` helper
- **Query cancellation**: per-tab context stored server-side, cancelled via tab ID
- **State split**: server state in TanStack Query, UI state in Zustand, persistent state in SQLite

## Gotchas

- oapi-codegen requires OpenAPI **3.0.3** (not 3.1.0)
- `CellValue` needs `x-go-type: '*string'` + `type: string, nullable: true`
- react-resizable-panels v4: exports are `Group`, `Panel`, `Separator`; use `orientation` not `direction`; strings like `"18%"` for percentages
- Storybook 10: don't export story names that shadow globals (e.g. `Array`)

# pglet

A single-binary PostgreSQL browser: Go backend (net/http + bbolt + lib/pq) with an embedded React SPA.

## Architecture

```
openapi.yaml (source of truth)
  ├─► oapi-codegen  → pkg/api/openapi.gen.go   (types + ServerInterface + embedded spec)
  └─► hey-api       → frontend/src/api/generated/ (TS types + SDK)

Frontend (React 19, Vite 7, Tailwind v4, TanStack Query v5, Monaco, Zustand)
  → pnpm build → frontend/dist/
  → go:embed   → single Go binary
```

### Backend layers

```
main.go          → CLI flags, http.Server, middleware chain, SPA handler
pkg/service/     → Business logic (single Service struct, methods namespaced by file)
pkg/api/         → Thin HTTP handlers (implement ServerInterface, call service)
pkg/client/      → PostgreSQL client wrapper (lib/pq)
pkg/repository/  → bbolt persistence (saved queries, history, settings, tab state)
pkg/ai/          → Anthropic API integration
```

Handlers receive `(w http.ResponseWriter, r *http.Request)`, parse the request, call `s.svc.Method()`, and write the response. All business logic lives in `pkg/service/`.

### Middleware stack

Applied in `main.go` via `http.Handler` wrapping:
1. **CORS** (dev/--cors only) — `api.CorsMiddleware`
2. **Recovery** — `api.RecoveryMiddleware`
3. **Logging** — `api.LoggingMiddleware`
4. **OpenAPI validation** — `nethttpmiddleware.OapiRequestValidator` (per-route via oapi-codegen)

## Project Layout

```
main.go                  # CLI flags, net/http setup, middleware, SPA fallback
openapi.yaml             # OpenAPI 3.0.3 spec (source of truth)
oapi-codegen.yaml        # codegen config (std-http-server + embedded-spec)
pkg/
  service/               # Business logic (single Service struct)
    service.go           # Struct, client management, errors, QueryError
    connection.go        # Connect, Disconnect, SwitchDatabase, ConnectionInfo, Databases
    schema.go            # Schemas, Objects, TableColumns, TableRows, TableInfo, ...
    query.go             # RunQuery, ExplainQuery, AnalyzeQuery, CancelQuery, ExportQuery
    saved.go             # ListSavedQueries, GetSavedQuery, Create/Update/DeleteSavedQuery
    history.go           # ListHistory, ClearHistory
    tabs.go              # GetTabState, SaveTabState
    ai.go                # GenerateSQL, AISuggestions, AITabName, HeuristicTabName
  api/                   # HTTP handlers (implement ServerInterface)
    server.go            # Server struct, NewServer()
    helpers.go           # writeJSON, readJSON
    middleware.go        # CorsMiddleware, LoggingMiddleware, RecoveryMiddleware
    handlers_connection.go
    handlers_schema.go
    handlers_query.go
    handlers_export.go
    handlers_saved.go
    handlers_history.go
    handlers_ai.go
    handlers_tabs.go
    openapi.gen.go       # Generated: types, ServerInterface, routing, embedded spec
  client/                # PostgreSQL client wrapper (lib/pq)
  repository/            # bbolt persistence (saved queries, history, settings)
  ai/                    # Anthropic API client (Claude Haiku for SQL generation)
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
make deps         # pnpm install + go mod tidy
```

Use `pnpm`, not npm.

## API Endpoints

| Group | Endpoints |
|-------|-----------|
| Connection | `POST /api/connect`, `POST /api/disconnect`, `POST /api/switchdb`, `GET /api/connection`, `GET /api/databases`, `GET /api/info` |
| Schema | `GET /api/schemas`, `GET /api/objects`, `GET /api/tables/{table}`, `GET /api/tables/{table}/rows\|info\|indexes\|constraints`, `GET /api/tables_stats`, `GET /api/server_settings`, `GET /api/activity` |
| Query | `POST /api/query`, `POST /api/explain`, `POST /api/analyze`, `POST /api/query/cancel` |
| Export | `POST /api/export` |
| Saved Queries | `GET\|POST /api/saved-queries`, `GET\|PUT\|DELETE /api/saved-queries/{id}` |
| History | `GET\|DELETE /api/history` |
| Tabs | `GET\|PUT /api/tabs` |
| AI | `POST /api/ai/generate`, `GET /api/ai/suggestions`, `POST /api/ai/tab-name` |

## Key Patterns

- **Spec-first**: always edit `openapi.yaml` first, then `make generate`
- **Service layer**: `pkg/service/Service` is a single struct with methods namespaced by file. Handlers never contain business logic.
- **Handler convention**: `pkg/api/handlers_<domain>.go` implements `ServerInterface` methods as thin HTTP wrappers
- **TanStack Query hooks**: `frontend/src/api/queries.ts` wraps generated SDK with `unwrap()` helper
- **Query cancellation**: per-tab context stored in Service, cancelled via tab ID
- **State split**: server state in TanStack Query, UI state in Zustand, persistent state in bbolt
- **OpenAPI validation**: incoming requests validated against embedded spec via `oapi-codegen/nethttp-middleware`

## Gotchas

- oapi-codegen requires OpenAPI **3.0.3** (not 3.1.0)
- `CellValue` needs `x-go-type: '*string'` + `type: string, nullable: true`
- `StdHTTPServerOptions.BaseURL` must be `""` not `"/"` (Go 1.22+ ServeMux rejects double-slash paths)
- react-resizable-panels v4: exports are `Group`, `Panel`, `Separator`; use `orientation` not `direction`; strings like `"18%"` for percentages
- Storybook 10: don't export story names that shadow globals (e.g. `Array`)

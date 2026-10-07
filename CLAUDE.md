# pglet

A single-binary PostgreSQL browser: Go backend (net/http + bbolt + lib/pq) with an embedded React SPA.

## Architecture

```
openapi.yaml (source of truth)
  ├─► oapi-codegen  → internal/api/openapi.gen.go   (types + ServerInterface + embedded spec)
  └─► hey-api       → frontend/src/api/generated/   (TS types + SDK)

Frontend (React 19, Vite 7, Tailwind v4, TanStack Query v5, Monaco, Zustand)
  → pnpm build → static/dist/   (committed so the binary builds without a frontend step)
  → go:embed   → single Go binary
```

## Layout

```
main.go, config.go, spa.go   # package main: CLI flags/config, http.Server + middleware, SPA fallback
internal/
  api/          # Thin HTTP handlers implementing ServerInterface (handlers_<domain>.go)
  service/      # Business logic: single Service struct, one file per domain
  client/       # PostgreSQL wrapper (lib/pq), one file per area: schema, table, tx, stats
  repository/   # bbolt persistence (saved queries, history, settings, tab state)
  ai/           # Anthropic API client
frontend/src/
  api/          # Generated SDK, TanStack Query hooks (queries.ts), queryClient
  components/   # One folder per feature; ui/ holds shared primitives
  stores/       # Zustand
  lib/          # Non-React helpers
static/         # go:embed entry point for the built frontend
dev/            # docker-compose + seed data for local Postgres
```

Endpoints are defined in `openapi.yaml`; don't duplicate them here.

## Dev Workflow

```bash
make dev          # Postgres + Vite :5173 + Go :8081
make generate     # Regen Go + TS from openapi.yaml
make build        # Production binary
make deps         # pnpm install + go mod tidy
cd frontend && pnpm ladle   # Component stories
```

Use `pnpm`, not npm.

## Key Patterns

- **Spec-first**: edit `openapi.yaml`, then `make generate`.
- **Handlers**: parse request → call one `s.svc.Method()` → write response. No business logic in `internal/api`.
- **Service**: new domain logic goes in `internal/service/<domain>.go` on the existing `Service` struct.
- **TanStack Query hooks**: `frontend/src/api/queries.ts` wraps the generated SDK with `unwrap()`.
- **State split**: server state in TanStack Query, UI state in Zustand, persistent state in bbolt.
- **Per-tab execution**: query cancellation and transactions are keyed by tab ID in the Service; `components/tabs/useTabExecution.ts` is the frontend side.
- **Middleware order** (`main.go`): CORS (dev/--cors only) → Recovery → Logging → OpenAPI validation (per-route).
- **Stories**: Ladle, `*.stories.tsx` next to the component. Global providers live in `frontend/.ladle/components.tsx`.

## Gotchas

- oapi-codegen requires OpenAPI **3.0.3** (not 3.1.0)
- `CellValue` needs `x-go-type: '*string'` + `type: string, nullable: true`
- `StdHTTPServerOptions.BaseURL` must be `""` not `"/"` (Go 1.22+ ServeMux rejects double-slash paths)
- react-resizable-panels v4: exports are `Group`, `Panel`, `Separator`; use `orientation` not `direction`; strings like `"18%"` for percentages
- Tailwind's `@import` in `index.css` pins `source("..")`; Ladle runs Vite from its own root, so auto-detection would find no classes
- Monaco custom themes must be defined in `beforeMount`, not `onMount`, or the editor starts on the fallback theme
- lib/pq's `BeginTx` sends `BEGIN READ WRITE` unless `TxOptions.ReadOnly` is set, which silently overrides the read-only session option; pass `ReadOnly: c.opts.ReadOnly` or use `Client.Begin`
- Integration tests in `internal/client` run only when `PGLET_TEST_DATABASE_URL` is set

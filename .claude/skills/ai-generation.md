# AI SQL Generation

## Overview

Natural language → SQL generation via Anthropic Claude API. The flow uses schema-aware prompting with an optional table pre-filtering step for large databases.

## Key Files

- `pkg/ai/generate.go` — Anthropic API calls, schema formatting, table selection
- `pkg/api/handlers_ai.go` — HTTP handler, schema introspection, orchestration
- `frontend/src/components/ai/AiQueryDialog.tsx` — Chat-style AI dialog
- `frontend/src/components/tabs/TabPanel.tsx` — Inline AI in query tabs
- `frontend/src/api/queries.ts` — `useAiGenerate()` mutation hook

## Sequence Diagram

```mermaid
sequenceDiagram
    participant User
    participant UI as React UI<br/>(TabPanel / AiQueryDialog)
    participant TQ as TanStack Query<br/>(useAiGenerate)
    participant SDK as hey-api SDK<br/>(aiGenerate)
    participant API as Go API Server<br/>(handlers_ai.go)
    participant Store as SQLite Store
    participant Client as PG Client
    participant PG as PostgreSQL
    participant Select as Anthropic API<br/>(Haiku – table selector)
    participant Gen as Anthropic API<br/>(Haiku – SQL generator)

    User->>UI: types natural language prompt
    UI->>TQ: mutate({ prompt, database, messages? })
    TQ->>SDK: POST /api/ai/generate
    SDK->>API: AiGenerate(req)

    API->>Store: GetSetting("ai_api_key")
    Store-->>API: apiKey

    alt API key missing
        API-->>SDK: 400 "AI API key not configured"
        SDK-->>TQ: error
        TQ-->>UI: show error
    end

    API->>Client: Objects()
    Client->>PG: query information_schema.tables, pg_matviews, pg_proc, ...
    PG-->>Client: schema objects
    Client-->>API: map[schema]*SchemaGroup

    loop each table in schema
        API->>Client: TableColumns(schema.table)
        Client->>PG: query information_schema.columns
        PG-->>Client: []Column
        Client-->>API: column info
    end

    Note over API: Build []TableSchema with column metadata

    alt schema has > 20 tables
        API->>Select: SelectTables(schema, prompt)
        Note over Select: System: "select relevant tables"<br/>User: table list + prompt
        Select->>Select: POST /v1/messages (Haiku)
        Select-->>API: JSON array of relevant table names
        Note over API: Filter schema to relevant tables
    end

    API->>Gen: Generate(prompt, filteredSchema, messages)
    Note over Gen: System: schema description + rules<br/>User: conversation history + prompt
    Gen->>Gen: POST /v1/messages (Haiku)
    Gen-->>API: JSON { sql, explanation }

    API-->>SDK: 200 { sql, explanation }
    SDK-->>TQ: AiGenerateResponse
    TQ-->>UI: onSuccess(data)
    UI->>UI: update editor with SQL,<br/>append AI turn to conversation
    UI-->>User: display generated SQL + explanation
```

## How It Works

1. **API key** is read from SQLite settings store (`ai_api_key`)
2. **Schema introspection** — `Client.Objects()` fetches all tables/views/functions from PostgreSQL, then `TableColumns()` gets column names and types for each table
3. **Table pre-filtering** (>20 tables only) — a cheap Haiku call receives just table names and picks the relevant subset, keeping the generation context small
4. **SQL generation** — filtered schema + conversation history + prompt sent to Haiku, which returns `{ sql, explanation }` as JSON
5. **Conversation support** — previous turns are passed as `messages[]` enabling multi-turn refinement

## Models

Both steps use **Claude Haiku 4.5** (`claude-haiku-4-5-20251001`):
- Table selection: system prompt asks for JSON array of relevant table names
- SQL generation: system prompt includes schema description + PostgreSQL rules

## Response Parsing

The generator expects JSON `{ sql, explanation }` but handles fallbacks:
1. Try direct JSON parse
2. Try extracting `{...}` from surrounding text
3. Fallback: treat entire response as raw SQL

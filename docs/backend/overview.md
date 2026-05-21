# Backend Architecture (Go API)

## Module

- **Path:** `apps/api`
- **Module name:** `codeatlas/apps/api`
- **Go version:** 1.23

## Binaries

| Command | Path | Purpose |
|---------|------|---------|
| `server` | `cmd/server/main.go` | HTTP API, migrations on boot, wires all services |
| `indexer` | `cmd/indexer/main.go` | CLI-only indexing (`-repo`, `-name`) |

## Boot sequence (`cmd/server/main.go`)

1. Load `config.Load()` from environment.
2. Connect `db.NewPool` → PostgreSQL.
3. Run `db.MigrateDir` on `MIGRATIONS_DIR`.
4. Register AI providers on `providers.Manager`.
5. Create `ai.NewRetriever(pool, socioStore)`.
6. Create `indexer.New(scanner, parser, postgresStore, logger)`.
7. Create `ingestion.NewService(socioStore, githubClient)`.
8. Create `repoingest.NewService(..., socioIngest, ...)`.
9. Start `httpserver.New` on `HTTP_ADDR` (default `:8080`).

## Package map

```
internal/
├── ai/                 # Chat, retrieval, prompts
│   └── providers/      # openai, local, anthropic, gemini, hf, openrouter
├── config/             # Environment configuration
├── db/                 # Pool + migration runner
├── github/             # GitHub REST client
├── graphhierarchy/     # Cluster layer builder
├── httpserver/         # HTTP routes + graph loaders
├── indexer/            # Parse, scan, resolve, persist
├── ingestion/          # Socio Phase 1 orchestration
├── llm/                # Low-level OpenAI/local HTTP clients
├── repoingest/         # Repository lifecycle
└── socio/              # Metrics, ownership, ingestion runs
```

## HTTP layer

- **Router:** Go 1.22+ `http.ServeMux` with method + path patterns (`GET /repositories/{id}/progress`).
- **Middleware:** `loggingMiddleware` (request logs), `withCORS` (origin allowlist).
- **No auth middleware.**

## Repository ingest (`repoingest`)

**Sources** (`internal/repoingest/types.go`):

| `sourceType` | Handler |
|--------------|---------|
| `github` | `GitSource` clone |
| `gitlab` | `GitSource` clone |
| `bitbucket` | `GitSource` clone |
| `zip` | `ZIPSource` extract |

**Status lifecycle:** `queued` → `cloning|extracting` → indexing substages → `ready|failed`.

**Workspace:** `WORKSPACE_ROOT` (default `./workspace`) — one folder per source type pattern; see `prepareWorkspacePath`.

**Post-ready:** `runSocioEnrichment` starts Phase 1 in a goroutine.

## Indexer (`indexer`)

Pipeline:

1. **Scan** TypeScript files (ignore `node_modules`, `dist`, `.git`, …).
2. **Parse** with Tree-sitter → symbols, imports, exports.
3. **Resolve** import paths to internal `files` rows where possible.
4. **Persist** `files`, `symbols`, `file_imports`, `file_exports`, `file_dependencies`.
5. **Embed** (optional) via `OpenAI` or local stub into `entity_embeddings`.

**Progress callbacks:** Used during `repoingest` background index to update `repositories` progress columns.

## Graph hierarchy (`graphhierarchy`)

Builds one **layer** for a path `prefix`:

- **Clusters:** immediate child folders with density metrics.
- **Files:** direct files under prefix.
- **Edges:** Aggregated dependency counts between cluster/file nodes.

Used exclusively by `GET /graph/clusters`.

## Error handling

- Handlers return JSON `{"error": "message"}` with appropriate HTTP status.
- Background ingest logs `repository_indexing_failed` and sets `status=failed` + `error_details`.
- Socio ingestion records failed steps in `socio_ingestion_steps.failure_metadata`.
- AI errors surface as `502` on chat or SSE `type: error`.

## Logging

- **Library:** `log/slog` JSON to stdout.
- **Examples:** `http_listening`, `graph_query_failed`, `socio_phase1_complete`, `repository_ingestion_ready`.

See: [indexer.md](./indexer.md), [ingestion-socio.md](./ingestion-socio.md), [ai-layer.md](./ai-layer.md).

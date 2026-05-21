# Indexing Pipeline

## Overview

The indexer transforms files on disk into a queryable **code graph** in PostgreSQL. It is invoked by:

- `repoingest` after clone/extract (primary path)
- `cmd/indexer` CLI for local debugging (`make index-repo REPO=/path`)

**Entry:** `internal/indexer/service.go` — `Run(ctx, Request)`.

## Components

| Component | File | Role |
|-----------|------|------|
| File scanner | `scanner.go` | Walk repo; filter TS files |
| Parser | `parser.go` | Tree-sitter AST → symbols/imports/exports |
| Resolver | `resolver.go` | Map module paths to file IDs |
| Postgres store | `postgres_store.go` | Upserts + embedding writes |
| Types | `types.go` | `Request`, `Result`, progress callback |

## Request shape

```go
type Request struct {
    RepositoryPath string
    RepositoryName string
    OnProgress     func(Progress) // optional; repoingest uses this
}
```

## Result metrics

Returned to caller:

- `Files`, `Symbols`, `FileDependencies`, `Embeddings`
- `Duration`

## Database writes

| Table | Content |
|-------|---------|
| `repositories` | Upsert by `root_path` / name |
| `files` | `relative_path` per repo |
| `symbols` | name, kind, line/col range |
| `file_imports` | module_path, external flag |
| `file_exports` | export names |
| `file_dependencies` | resolved edges between files |
| `entity_embeddings` | vectors for file/symbol entities |

## Embeddings

- **With `OPENAI_API_KEY`:** `llm.NewOpenAIClient` + `text-embedding-3-small` (configurable).
- **Without key:** `llm.NewLocalClient` — deterministic stub vectors (semantic search quality is not production-grade).

**Why embeddings matter:** `ai.Retriever.semanticFileSeeds` orders files by cosine similarity to the user query.

## Ignored paths (typical)

Implemented in scanner—includes `node_modules`, `dist`, `build`, `.git`, coverage dirs (see `scanner.go` for full list).

## Tests

| File | Coverage |
|------|----------|
| `parser_test.go` | Tree-sitter parse samples |
| `resolver_test.go` | Import path resolution |

## Limitations

- **TypeScript only** in MVP scanner/parser pairing.
- **Single repository workspace** per ingest job under `WORKSPACE_ROOT`.
- **No incremental index** — reindex replaces graph for that repository id.

## Operations

```bash
# From repo root
make index-repo REPO=/absolute/path/to/typescript-repo

# Equivalent
pnpm --filter @codeatlas/api index -- -repo "/path" -name "my-repo"
```

Ensure `DATABASE_URL` is set and migrations have run (API server boot applies them).

# Indexing Pipeline

## Overview

The indexer transforms files on disk into a queryable **code graph** in PostgreSQL. It is invoked by:

- `repoingest` after clone/extract (primary path)
- `cmd/indexer` CLI for local debugging (`make index-repo REPO=/path`)

**Entry:** `internal/indexer/service.go` — `Run(ctx, Request)`.

## Components

| Component | File | Role |
|-----------|------|------|
| File scanner | `scanner.go` | Walk repo; index supported source extensions |
| Parser | `parser_cgo.go` / `parser_nocgo.go` | Tree-sitter (CGO) or regex fallback → symbols/imports/exports |
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
| `parser_test.go` | Tree-sitter / parse-bytes samples |
| `language_test.go` | Extension → language routing |
| `fallback_test.go` | Regex fallback + multi-language scan |
| `resolver_test.go` | Import path resolution |

## Supported languages

Extensions are mapped in `language.go` and parsed in `parser_cgo.go` (Tree-sitter grammars via `github.com/tree-sitter/go-tree-sitter`) or `fallback_regex.go` when `CGO_ENABLED=0`.

| Language | Extensions (examples) |
|----------|---------------------|
| TypeScript | `.ts`, `.tsx`, `.mts`, `.cts` |
| JavaScript | `.js`, `.jsx`, `.mjs`, `.cjs` |
| Python | `.py` |
| Go | `.go` |
| Java | `.java` |
| C | `.c`, `.h` |
| C++ | `.cpp`, `.cc`, `.hpp`, … |
| PHP | `.php` |
| C# | `.cs` |

Production Docker builds set `CGO_ENABLED=1` in `apps/api/Dockerfile`.

## Limitations

- **Heuristic import resolution** — local path resolution works best for relative imports (TS/JS, Go); Java/C# package imports are stored but not always resolved to files.
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

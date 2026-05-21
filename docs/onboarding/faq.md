# FAQ

## What languages does CodeAtlas index?

**TypeScript** in the current MVP (`internal/indexer` Tree-sitter TS parser). Other languages require new scanner/parser pairs.

## Is the Python AI service required?

**No.** Chat runs in the Go API (`POST /ai/chat`). `apps/ai` only exposes `GET /health` today.

## Why is socio sync skipped?

Common reasons (see `ingestion/service.go`):

- Repository `sourceType` is not `github` (gitlab/bitbucket/zip skip Phase 1)
- `GITHUB_TOKEN` is not set in the API environment

## Does CodeAtlas have user accounts?

**No.** There is no login or per-user authorization in the codebase.

## How is the graph different from GitHub Insights?

GitHub shows repo activity; CodeAtlas merges **dependency structure**, **symbols**, **embeddings**, and **file-level metrics** for architecture Q&A and map navigation in one product.

## Can I use GitLab or Bitbucket?

**Clone/index:** Yes (`sourceType` gitlab/bitbucket).  
**Socio Phase 1:** No—GitHub REST only.

## What embeddings model is used?

Default `text-embedding-3-small` via OpenAI when `OPENAI_API_KEY` is set. Configurable via `OPENAI_EMBEDDING_MODEL`.

## What happens on delete repository?

`DELETE /repositories/{id}` removes DB rows (CASCADE) and workspace files. Response includes `undo` metadata to re-submit the same source URL manually.

## Is streaming chat supported?

Yes—`"stream": true` on `POST /ai/chat` returns SSE `token` events.

## Where are migrations applied?

Automatically when the API server starts (`db.MigrateDir` in `main.go`). CLI indexer does not migrate—start API once or run server before index-only workflows.

## What's planned but not built?

- Phase 2: PR/issue comment signal extraction → `architecture_signals`
- Phase 3: CI run ingestion → operational risk APIs
- `GET /repositories/{id}/risk-summary` (not in httpserver)
- Production deploy manifests and CI workflows

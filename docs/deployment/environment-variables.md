# Environment Variables

Variables read by the codebase (from `config.Load`, Vite, Compose, or Pydantic settings).

## Root `.env` (Docker Compose)

| Variable | Default | Used by |
|----------|---------|---------|
| `POSTGRES_USER` | `codeatlas` | `docker-compose.yml` |
| `POSTGRES_PASSWORD` | `codeatlas` | Compose |
| `POSTGRES_DB` | `codeatlas` | Compose |
| `POSTGRES_PORT` | `5432` | Compose port mapping |
| `DATABASE_URL` | — | Convenience for local tools |
| `HTTP_ADDR` | — | Documentation |
| `CORS_ALLOWED_ORIGINS` | — | Documentation |
| `VITE_API_URL` | — | Documentation |

## `apps/api` (Go server)

| Variable | Default | Purpose |
|----------|---------|---------|
| `HTTP_ADDR` | `:8080` | Listen address |
| `DATABASE_URL` | — | **Required** Postgres DSN |
| `CORS_ALLOWED_ORIGINS` | localhost:5173 | Comma-separated origins |
| `MIGRATIONS_DIR` | `./migrations` | SQL migrations path |
| `WORKSPACE_ROOT` | `./workspace` | Clone/extract directory |
| `ZIP_MAX_BYTES` | `104857600` | Max ZIP upload (100MB) |
| `ZIP_MAX_FILES` | `5000` | Max files in ZIP |
| `OPENAI_API_KEY` | — | Embeddings + OpenAI provider |
| `OPENAI_CHAT_MODEL` | `gpt-4o-mini` | Chat model |
| `OPENAI_EMBEDDING_MODEL` | `text-embedding-3-small` | Embedding model |
| `AI_DEFAULT_PROVIDER` | `local` | Default chat provider |
| `AI_DEFAULT_MODEL` | `local-default` | Default model name |
| `AI_CONTEXT_TOKEN_BUDGET` | `7000` | Prompt size cap |
| `GITHUB_TOKEN` | — | GitHub REST for socio Phase 1 |
| `ANTHROPIC_API_KEY` | — | Provider registration |
| `GEMINI_API_KEY` | — | Provider registration |
| `HUGGINGFACE_API_KEY` | — | Provider registration |
| `OPENROUTER_API_KEY` | — | Provider registration |

**Source:** `apps/api/internal/config/config.go`

## `apps/web` (Vite)

| Variable | Default | Purpose |
|----------|---------|---------|
| `VITE_API_URL` | `http://localhost:8080` | Baked into client (optional) |
| `VITE_API_PROXY_TARGET` | `http://localhost:8080` | Dev proxy target |

**Source:** `apps/web/vite.config.ts`, `apps/web/src/apiBase.ts`

## `apps/ai` (Python)

| Variable | Purpose |
|----------|---------|
| `CODEATLAS_AI_*` | Prefix for Pydantic settings (e.g. database URL) |

**Source:** `apps/ai/.env.example` — service not wired to main product flow.

## Examples

**Minimal local API (`apps/api/.env`):**

```env
DATABASE_URL=postgresql://codeatlas:codeatlas@localhost:5432/codeatlas
```

**Full local with AI + socio:**

```env
DATABASE_URL=postgresql://codeatlas:codeatlas@localhost:5432/codeatlas
OPENAI_API_KEY=sk-...
GITHUB_TOKEN=ghp_...
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

**Never commit** real keys to git. Use `.env` (gitignored) or a secret manager in production.

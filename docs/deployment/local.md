# Local Development

## Prerequisites

| Tool | Version |
|------|---------|
| Node.js | 20+ |
| pnpm | 9+ |
| Go | 1.23+ |
| Docker + Compose | For Postgres |
| Python | 3.11+ (optional, `apps/ai`) |
| Make | Optional (Git Bash/WSL on Windows) |

## First-time setup

```bash
# 1. Clone and install JS deps
make install
# or: pnpm install

# 2. Environment files
cp .env.example .env
cp apps/api/.env.example apps/api/.env
cp apps/web/.env.example apps/web/.env.local

# 3. Start Postgres
make docker-up

# 4. (Optional) Python AI stub
make ai-sync

# 5. Run all dev processes
make dev
```

## Running services individually

```bash
# Database only
docker compose up -d postgres

# API (from apps/api, migrations run on start)
cd apps/api
set DATABASE_URL=postgresql://codeatlas:codeatlas@localhost:5432/codeatlas
go run ./cmd/server

# Web
pnpm --filter @codeatlas/web dev

# CLI index only
make index-repo REPO=C:\path\to\ts-repo
```

## Default URLs

| Service | URL |
|---------|-----|
| Web | http://localhost:5173 |
| API health | http://localhost:8080/health |
| Python AI | http://localhost:8001/health |
| Postgres | localhost:5432 |

## Optional: GitHub socio sync

Add to `apps/api/.env`:

```env
GITHUB_TOKEN=ghp_your_token_with_repo_read
```

Re-index or ingest a **GitHub** repository. Poll:

```bash
curl http://localhost:8080/repositories/1/ingestion/status
```

## Optional: OpenAI

```env
OPENAI_API_KEY=sk-...
```

Enables real embeddings during index and OpenAI chat provider.

## Windows notes

- `Makefile` targets expect **Make** (Git Bash/WSL) or run underlying commands manually.
- Use PowerShell env: `$env:DATABASE_URL="postgresql://..."`
- Path with spaces (e.g. `New folder`) is supported but quote paths in CLI args.

## Workspace data

- Cloned repos: `apps/api/workspace/` (gitignored)
- Do not commit tokens in `.env` files

## Quality commands

```bash
make lint
make typecheck
make format
make lint-ruff   # apps/ai only
```

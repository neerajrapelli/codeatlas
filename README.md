# CodeAtlas

CodeAtlas is an **architecture intelligence** platform: it ingests repositories, builds a continuously updating **semantic graph**, and exposes that graph through APIs and a minimal UI surface.

This repository is a **pnpm + Turborepo monorepo** with a Go API, a Python AI service, shared TypeScript packages, and a Vite + React frontend.

## Repository layout

```
apps/
  web/        React + TypeScript (Vite)
  api/        Go HTTP API
  ai/         Python AI service (FastAPI)

packages/
  shared-types/   Cross-layer types (API contracts, graph primitives)
  graph-core/     Pure TypeScript graph helpers (used by the web surface first)

infra/
  docker/         Local dependency initialization (Postgres extensions, etc.)
```

## Prerequisites

- **Node.js** 20+ and **pnpm** 9+
- **Go** 1.22+
- **Python** 3.11+ (for `apps/ai`)
- **Docker** + Docker Compose (for local Postgres + pgvector)

## Quick start

### 1) Install JavaScript dependencies

```bash
make install
```

### 2) Environment variables

Copy the example env file and adjust as needed:

```bash
cp .env.example .env
cp apps/web/.env.example apps/web/.env.local
```

Notes:

- **Root `.env`** is used by **Docker Compose** (and is a convenient place to store local `DATABASE_URL` defaults).
- **`apps/web/.env.local`** is loaded by Vite for `VITE_*` variables.
- **`apps/api`** can read from your shell environment or an `apps/api/.env` file if you use a local loader; see `apps/api/.env.example`.
- **`apps/ai`** loads `apps/ai/.env` via Pydantic settings; see `apps/ai/.env.example`.

### 3) Start Postgres (pgvector)

Docker Compose reads a root `.env` file automatically (if present) for variable interpolation. If you have not created `.env` yet, the compose file falls back to the documented defaults.

```bash
make docker-up
```

### 4) Install Python tooling (recommended)

This installs Ruff + service dependencies for local lint/dev:

```bash
make ai-sync
```

### 5) Run the whole dev stack

```bash
make dev
```

Useful Turborepo filters:

```bash
pnpm dev --filter @codeatlas/web
pnpm dev --filter @codeatlas/api
pnpm dev --filter @codeatlas/ai
```

## Common commands

```bash
make build
make lint
make format
make typecheck
make clean
```

Python formatting/linting (optional, but recommended):

```bash
make ai-sync
make lint-ruff
```

Repository ingestion (TypeScript repos only, first MVP pipeline):

```bash
make index-repo REPO=/absolute/path/to/your/typescript-repo
```

This runs `apps/api/cmd/indexer`, which:
- recursively scans files (ignoring `node_modules`, `dist`, `build`, `.git`)
- parses TypeScript via Tree-sitter
- extracts imports/exports/functions/classes/interfaces
- writes entities and graph relationships into PostgreSQL
- optionally generates embeddings for files/symbols/import relationships when `OPENAI_API_KEY` is configured

AI architecture chat:

```bash
curl -X POST http://localhost:8080/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"repositoryId":1,"query":"What breaks if I change auth?"}'
```

Chat uses graph-aware retrieval (semantic + dependency expansion) and only sends compact retrieved context to the LLM.

Repository onboarding API:

```bash
# Git-based sources (github/gitlab/bitbucket)
curl -X POST http://localhost:8080/repositories \
  -H "Content-Type: application/json" \
  -d '{"sourceType":"github","sourceUrl":"https://github.com/octokit/types.ts.git","branch":"main"}'

# ZIP upload
curl -X POST http://localhost:8080/repositories \
  -F "sourceType=zip" \
  -F "displayName=my-zip-repo" \
  -F "file=@/path/to/repo.zip"
```

Status lifecycle: `queued` → `cloning|extracting` → `indexing` → `ready|failed`.

## Local URLs (defaults)

- **Web**: `http://localhost:5173`
- **Go API**: `http://localhost:8080/health`
- **AI service**: `http://localhost:8001/health`
- **Postgres**: `localhost:5432`

## Makefile + Windows

`Makefile` targets are written for **Make** (commonly used via **Git Bash** or **WSL** on Windows). If you do not have Make installed, use the underlying `pnpm` / `docker compose` commands shown in the `Makefile`.

## License

Proprietary (update when you publish).

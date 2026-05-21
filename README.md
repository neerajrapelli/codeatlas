# CodeAtlas

**Architecture intelligence for software teams** — ingest repositories, build a semantic dependency graph, enrich it with socio-technical signals (ownership, churn, hotspots), and explore impact through an interactive map and graph-grounded AI.

> **Full documentation:** [docs/README.md](docs/README.md) · [ARCHITECTURE.md](ARCHITECTURE.md) · [API_REFERENCE.md](API_REFERENCE.md)

## What it does

| Capability | Status |
|------------|--------|
| Ingest GitHub / GitLab / Bitbucket / ZIP | ✅ |
| Multi-language parsing (TS/JS, Go, Python, Java, C/C++, PHP, C#) | ✅ Tree-sitter (CGO) + regex fallback |
| Dependency graph + cluster map UI | ✅ |
| Semantic embeddings + Graph RAG chat (SSE) | ✅ (OpenAI recommended) |
| GitHub history → ownership & hotspots | ✅ (`GITHUB_TOKEN`) |
| Engineering memory / CI risk (Phase 2–3) | Schema only |

**Production notes:** JWT auth and Docker Compose are available; set `AUTH_DISABLED=false` and a strong `JWT_SECRET` for real deployments. CI workflows are not yet in-repo.

## Stack

- **Monorepo:** pnpm + Turborepo
- **API:** Go 1.23, PostgreSQL, pgvector
- **Web:** React 19, Vite, React Flow, ELK
- **AI:** In-process Go providers (`internal/ai`); Python `apps/ai` is a health stub

> **Note on apps/ai:** The Python AI service is a stub and is NOT required for local development.
> Run `make dev` without it. Only start it if working on evaluation features.
> See [apps/ai/README.md](apps/ai/README.md) for details.

## Quick start

```bash
make install
cp .env.example .env
cp apps/api/.env.example apps/api/.env
make docker-up
make dev
```

| Service | URL |
|---------|-----|
| Web UI | http://localhost:5173 |
| API | http://localhost:8080/health |
| Postgres | localhost:5432 |

Optional:

```env
# apps/api/.env
OPENAI_API_KEY=sk-...      # embeddings + chat
GITHUB_TOKEN=ghp_...       # socio-technical sync
```

## Repository layout

```
apps/
  api/     Go HTTP API + indexer CLI
  web/     React workspace
  ai/      Python FastAPI (health only)
packages/
  shared-types/   API contracts
  graph-core/     Graph helpers
docs/             Professional documentation set
```

## Example API calls

```bash
# Ingest
curl -X POST http://localhost:8080/repositories \
  -H "Content-Type: application/json" \
  -d '{"sourceType":"github","sourceUrl":"https://github.com/org/repo","branch":"main"}'

# Architecture chat
curl -X POST http://localhost:8080/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"repositoryId":1,"query":"What breaks if we change auth?"}'
```

## Documentation index

| Topic | Link |
|-------|------|
| Architecture & diagrams | [docs/architecture/overview.md](docs/architecture/overview.md) |
| Backend | [docs/backend/overview.md](docs/backend/overview.md) |
| Frontend | [docs/frontend/overview.md](docs/frontend/overview.md) |
| Database | [docs/database/schema.md](docs/database/schema.md) |
| API | [docs/api/endpoints.md](docs/api/endpoints.md) |
| Security | [docs/security/overview.md](docs/security/overview.md) |
| Local dev | [docs/deployment/local.md](docs/deployment/local.md) |
| Onboarding | [docs/onboarding/developer-guide.md](docs/onboarding/developer-guide.md) |
| Troubleshooting | [TROUBLESHOOTING.md](TROUBLESHOOTING.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |

## Common commands

```bash
make dev          # Turbo: web + api + packages
make build
make lint
make typecheck
make index-repo REPO=/path/to/typescript-repo
make docker-up
```

## Windows

Use **Git Bash**, **WSL**, or run `pnpm` / `docker compose` commands from the Makefile manually. See [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## License

Proprietary (update when published).

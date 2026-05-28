# CodeAtlas Documentation

Professional documentation for the CodeAtlas architecture intelligence platform. This index is intended for engineers onboarding to the monorepo or operating it in production.

## What is CodeAtlas?

CodeAtlas ingests software repositories, builds a **semantic dependency graph** in PostgreSQL, enriches it with **socio-technical signals** (ownership, churn, hotspots), and exposes the result through a **Go HTTP API** and a **React** workspace for exploration and AI-assisted architecture Q&A.

The graph is the **source of truth**—features are graph enrichment, not disconnected GitHub widgets.

## Documentation map

| Topic | Path |
|-------|------|
| System overview & diagrams | [architecture/overview.md](./architecture/overview.md) |
| Data flows | [architecture/data-flow.md](./architecture/data-flow.md) |
| Design decisions | [architecture/decisions.md](./architecture/decisions.md) |
| Sequence diagrams | [architecture/sequence-diagrams.md](./architecture/sequence-diagrams.md) |
| Performance & scaling | [architecture/performance-and-scaling.md](./architecture/performance-and-scaling.md) |
| Runtime tracing | [architecture/runtime-tracing.md](./architecture/runtime-tracing.md) |
| Backend (Go API) | [backend/overview.md](./backend/overview.md) |
| Indexing pipeline | [backend/indexer.md](./backend/indexer.md) |
| Socio-technical ingestion | [backend/ingestion-socio.md](./backend/ingestion-socio.md) |
| AI / RAG layer | [backend/ai-layer.md](./backend/ai-layer.md) |
| Frontend | [frontend/overview.md](./frontend/overview.md) |
| Database schema | [database/schema.md](./database/schema.md) |
| Entity relationships (ERD) | [database/erd.md](./database/erd.md) |
| HTTP API reference | [api/endpoints.md](./api/endpoints.md) |
| Environment variables | [deployment/environment-variables.md](./deployment/environment-variables.md) |
| Local setup | [deployment/local.md](./deployment/local.md) |
| Production assumptions | [deployment/production.md](./deployment/production.md) |
| Security | [security/overview.md](./security/overview.md) |
| Testing | [testing/strategy.md](./testing/strategy.md) |
| Architecture dependency checks | [architecture/dependency-analysis.md](./architecture/dependency-analysis.md) |
| ADR index | [adr/README.md](./adr/README.md) |
| Logging & monitoring | [operations/logging-monitoring.md](./operations/logging-monitoring.md) |
| Prometheus metrics | [operations/observability.md](./operations/observability.md) |
| Troubleshooting | [operations/troubleshooting.md](./operations/troubleshooting.md) |
| Developer onboarding | [onboarding/developer-guide.md](./onboarding/developer-guide.md) |
| FAQ | [onboarding/faq.md](./onboarding/faq.md) |

## Root-level guides (repo root)

- [../README.md](../README.md) — Quick start
- [../ARCHITECTURE.md](../ARCHITECTURE.md) — Architecture summary
- [../API_REFERENCE.md](../API_REFERENCE.md) — API quick reference
- [../DEPLOYMENT.md](../DEPLOYMENT.md) — Deployment guide
- [../TROUBLESHOOTING.md](../TROUBLESHOOTING.md) — Common issues
- [../CONTRIBUTING.md](../CONTRIBUTING.md) — Contribution workflow

## Technology stack (as implemented)

| Layer | Technology |
|-------|------------|
| Monorepo | pnpm workspaces + Turborepo |
| API | Go 1.23, `net/http`, pgx, pgvector |
| Parsing | Tree-sitter (TypeScript MVP) |
| DB | PostgreSQL 16 + pgvector + pgcrypto |
| Web | React 19, TypeScript, Vite 6, React Flow, ELK |
| AI (in-process) | Provider manager in Go (`internal/ai`) |
| AI (stub service) | Python FastAPI `apps/ai` (health only) |
| Local infra | Docker Compose (Postgres only) |

## Maturity notes

- **Implemented:** Code indexing, graph UI, Graph RAG chat (SSE), repo ingest (Git + ZIP), Phase 1 socio-technical sync (GitHub + `GITHUB_TOKEN`).
- **Schema only / partial:** Phase 2 engineering memory tables, Phase 3 CI tables—migrations exist; ingestion not fully wired.
- **Not in repo:** User authentication, production Docker/K8s manifests.

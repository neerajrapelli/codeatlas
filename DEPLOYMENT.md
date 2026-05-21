# Deployment Guide

> **Note:** Production deploy configs are **not** in this repository. See [docs/deployment/production.md](docs/deployment/production.md) for assumptions and recommended topology.

## Local (supported)

```bash
make install
cp .env.example .env
make docker-up
make dev
```

Details: [docs/deployment/local.md](docs/deployment/local.md)

## Production checklist (summary)

1. **Postgres 16+** with `vector` and `pgcrypto`
2. **Run migrations** before traffic (`apps/api/migrations`)
3. **Deploy Go API** with secrets: `DATABASE_URL`, `CORS_ALLOWED_ORIGINS`, optional `OPENAI_API_KEY`, `GITHUB_TOKEN`
4. **Deploy web** static build from `pnpm --filter @codeatlas/web build`
5. **Writable volume** for `WORKSPACE_ROOT`
6. **Place auth** at API gateway (required—no in-app auth)
7. **Configure TLS** and backups

## Build artifacts

```bash
# API binary
cd apps/api && go build -o bin/server ./cmd/server

# Web static
pnpm --filter @codeatlas/web build
# Output: apps/web/dist/
```

## CI/CD

No workflows ship with the repo. Suggested pipeline documented in [docs/deployment/production.md](docs/deployment/production.md).

## Health checks

- API: `GET /health`
- DB: `pg_isready` (Compose healthcheck pattern in `docker-compose.yml`)

# Developer Onboarding Guide

Welcome to CodeAtlas. This guide gets you from zero to a working local workspace in under 30 minutes.

## Day 1: Environment

1. Install Node 20+, pnpm 9+, Go 1.23+, Docker.
2. Clone the repository.
3. Run:

```bash
make install
cp .env.example .env
cp apps/api/.env.example apps/api/.env
make docker-up
make dev
```

4. Open http://localhost:5173
5. Ingest a small public TypeScript repo (example in UI: `simple-typescript-starter`).

## Day 2: Code tour

Read in this order:

| Order | Document / code |
|-------|-----------------|
| 1 | [architecture/overview.md](../architecture/overview.md) |
| 2 | `apps/api/cmd/server/main.go` |
| 3 | `apps/api/internal/repoingest/service.go` |
| 4 | `apps/api/internal/indexer/service.go` |
| 5 | `apps/web/src/App.tsx` |
| 6 | `apps/api/internal/ai/retriever.go` |

## Day 3: Make a small change

Suggested first tasks:

- Add a field to `GET /health` (version from git sha)
- Add one unit test in `internal/socio`
- Improve empty state copy in `SocioPanels.tsx`

Run before PR:

```bash
make lint
make typecheck
cd apps/api && go test ./...
```

## Conventions

| Topic | Convention |
|-------|------------|
| Go packages | `internal/` only; no exports outside module |
| HTTP routes | Go 1.22 path patterns in `httpserver` |
| DB changes | New `000N_name.up.sql` + paired `.down.sql` |
| Frontend | Functional components, hooks, no global store |
| Logs | `slog` structured keys, snake_case |

## Monorepo commands

```bash
pnpm dev --filter @codeatlas/web    # UI only
pnpm dev --filter @codeatlas/api    # API only
make index-repo REPO=/path          # CLI index
```

## Getting help

- [FAQ](./faq.md)
- [TROUBLESHOOTING.md](../../TROUBLESHOOTING.md)
- [API endpoints](../api/endpoints.md)

## What not to do yet

- Do not assume Phase 2/3 APIs exist—check `httpserver` first.
- Do not add auth only in frontend—must be server-side.
- Do not store GitHub tokens in `VITE_*` env vars.

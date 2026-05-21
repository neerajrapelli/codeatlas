# Contributing to CodeAtlas

Thank you for improving CodeAtlas. This project prioritizes **graph-first**, **incremental**, and **observable** changes over large rewrites.

## Before you start

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) and [docs/onboarding/developer-guide.md](docs/onboarding/developer-guide.md).
2. Set up locally per [docs/deployment/local.md](docs/deployment/local.md).

## Development workflow

```bash
make install
make docker-up
make dev
```

For focused work:

```bash
pnpm dev --filter @codeatlas/api
pnpm dev --filter @codeatlas/web
```

## Quality gates

```bash
make lint
make typecheck
cd apps/api && go test ./...
```

Optional: `make lint-ruff` for Python.

## Pull request guidelines

| Area | Expectation |
|------|-------------|
| Scope | One logical change; avoid drive-by refactors |
| Migrations | Include `*.up.sql` and `*.down.sql`; never edit applied migrations in prod |
| APIs | Update [docs/api/endpoints.md](docs/api/endpoints.md) |
| UI | Match existing `styles.css` patterns |
| Secrets | Never commit `.env`, keys, or tokens |
| Tests | Add tests for non-trivial Go logic |
| Docs | Update relevant `docs/` page when behavior changes |

## Architecture rules (product)

1. **Enrich the graph** — new signals should attach to `files` / `repositories`, not parallel silos.
2. **Progressive enrichment** — do not block the map on slow optional pipelines.
3. **Observability** — ingestion steps should log and persist duration/counts where applicable.
4. **No placeholder TODOs** in production paths—ship complete behavior or explicit `skipped` status.

## Code style

- **Go:** standard `gofmt`, explicit error handling, `slog` for logs
- **TypeScript:** ESLint + Prettier via repo config
- **Commits:** clear subject; explain *why* in body

## Reporting issues

Include: OS, command run, API log snippet, `repositoryId`, `status`, and `errorDetails` if ingest-related.

## License

Proprietary until published—confirm with maintainers before external distribution.

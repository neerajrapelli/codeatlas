# Troubleshooting (Operations)

See also root [TROUBLESHOOTING.md](../../TROUBLESHOOTING.md).

## API will not start

| Symptom | Cause | Fix |
|---------|-------|-----|
| `db_connect_failed` | Postgres down / wrong URL | `make docker-up`, verify `DATABASE_URL` |
| `db_migrate_failed` | SQL error | Check logs; fix migration; `psql` inspect `schema_migrations` |
| Port in use | `:8080` taken | Change `HTTP_ADDR` or stop other process |

## Web shows "Network request failed"

| Cause | Fix |
|-------|-----|
| API not running | Start `go run ./cmd/server` |
| Wrong API URL | Use dev proxy: leave `VITE_API_URL` unset; use http://localhost:5173 |
| CORS | Add your origin to `CORS_ALLOWED_ORIGINS` if bypassing proxy |

## Repository stuck in queued / indexing

1. `GET /repositories/{id}/progress`
2. Check API logs for `repository_indexing_failed`
3. Verify disk space under `WORKSPACE_ROOT`
4. For private GitHub repos, ensure clone credentials (HTTPS URL + machine git creds—not fully automated in code)

## Index completes but map empty

- `filesIndexed` must be > 0
- TypeScript files required for MVP indexer
- Check repo actually contains `.ts` sources (not only `.js`)

## Socio data missing

| Check | Expected |
|-------|----------|
| `sourceType` | `github` |
| `GITHUB_TOKEN` | Set in API env |
| `GET .../ingestion/status` | `socioTechnical.status` = `completed` or `skipped` |

If `skipped` with message about token, set `GITHUB_TOKEN` and reindex.

## AI chat returns empty / stub

- No embeddings: set `OPENAI_API_KEY` and reindex
- `ContextFileCount == 0`: repository not indexed or query unrelated to indexed paths
- Provider `local`: returns placeholder-style answers

## SSE stream stops mid-answer

- Proxy buffering: disable buffering on nginx for `text/event-stream`
- Timeout: API uses 120s context for stream handler

## Docker Postgres unhealthy

```bash
docker compose logs postgres
make db-shell
```

Verify extension: `\dx` should list `vector`.

## Windows path issues

Quote paths with spaces; run API from `apps/api` so `MIGRATIONS_DIR=./migrations` resolves.

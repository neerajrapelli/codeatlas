# Logging and Monitoring

## Logging (implemented)

### Backend

- **Library:** Go `log/slog` with **JSON** handler to stdout (`cmd/server/main.go`).
- **HTTP:** `loggingMiddleware` logs method, path, status, duration.

### Representative log keys

| Message | When |
|---------|------|
| `http_listening` | Server start |
| `db_connect_failed` / `db_migrate_failed` | Boot failure |
| `repository_ingestion_ready` | Code index complete |
| `repository_indexing_failed` | Index error |
| `socio_phase1_complete` | GitHub socio done |
| `socio_ingestion_skipped` | Non-GitHub or no token |
| `socio_ingestion_failed` | Socio error |
| `graph_query_failed` | DB/graph errors |
| `ai_chat_failed` | Chat errors |

### Ingestion DB audit

Query observability without log access:

```sql
SELECT phase, status, completion_percent, error_details, started_at, completed_at
FROM socio_ingestion_runs
WHERE repository_id = 1
ORDER BY created_at DESC
LIMIT 5;

SELECT step, status, duration_ms, items_processed, failure_metadata
FROM socio_ingestion_steps
WHERE run_id = '...';
```

Repository code index progress: `repositories.current_stage`, `progress_percent`, `stage_metadata` JSON.

## Metrics (not implemented in code)

**Assumption:** Production should add Prometheus/OpenTelemetry:

| Metric | Type | Labels |
|--------|------|--------|
| `http_request_duration_seconds` | histogram | route, status |
| `index_files_total` | counter | repository_id |
| `socio_ingestion_duration_seconds` | histogram | phase, status |
| `github_rate_limit_remaining` | gauge | — |
| `ai_chat_tokens` | counter | provider |

## Monitoring checklist

- [ ] Health check: `GET /health` every 30s
- [ ] DB connectivity alert
- [ ] Disk usage on `WORKSPACE_ROOT`
- [ ] Failed repository count (`status=failed`)
- [ ] Socio runs stuck in `running` > N hours

## Tracing

No distributed tracing in codebase. For multi-service future (Python AI), propagate `traceparent` at gateway.

## Frontend errors

- Network failures: caught in `fetch`; user sees warning strings
- No Sentry integration in repo

## Dashboard ideas

1. Repositories by status (ready / indexing / failed)
2. Median time-to-ready
3. Top hotspot files per org
4. AI chat error rate

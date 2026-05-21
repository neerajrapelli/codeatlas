# Observability

## Prometheus metrics

The API exposes `GET /metrics` (unauthenticated by default; restrict at the reverse proxy in production).

| Metric | Description |
|--------|-------------|
| `codeatlas_http_requests_total` | Counter by method, route template, status |
| `codeatlas_http_request_duration_seconds` | Request latency histogram |
| `codeatlas_ingestion_jobs_active` | Gauge (reserved for future job polling) |

### Scraping example

```yaml
scrape_configs:
  - job_name: codeatlas-api
    static_configs:
      - targets: ['api:8080']
    metrics_path: /metrics
```

## Logs

The API emits JSON logs via `slog` (`http_request` per request in `loggingMiddleware`).

## Rate limiting

Set `REDIS_URL` so all API replicas share per-IP limits for `POST /repositories`, `POST .../reindex`, and `POST /ai/chat`. Without Redis, an in-memory limiter is used (single replica only).

## Alerting suggestions

- `codeatlas_http_requests_total{status="503"}` on `/health` — database down
- High `429` rate — abuse or tight limits
- Ingestion queue depth (add custom metric when job metrics are wired)

# Runtime Tracing

CodeAtlas API already includes OpenTelemetry hooks for HTTP and Postgres spans.

## Enable tracing locally

Set these environment variables for `apps/api`:

```env
OTEL_SERVICE_NAME=codeatlas-api
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_INSECURE=true
OTEL_SDK_DISABLED=false
```

If no OTLP endpoint is configured, traces are emitted to stdout for debugging.

## Docker Compose support

`docker-compose.yml` includes an optional Jaeger all-in-one collector + UI:

- OTLP HTTP ingest: `http://localhost:4318`
- UI: [http://localhost:16686](http://localhost:16686)

The API service can send traces directly to that endpoint through `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Instrumentation touchpoints

- HTTP middleware via `internal/telemetry.HTTPMiddleware(...)`
- DB client spans via `internal/telemetry.PGXTracer()`
- Manual spans via `internal/telemetry.Start(...)`

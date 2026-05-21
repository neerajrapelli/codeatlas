# syntax=docker/dockerfile:1

# --- Build (CGO required for tree-sitter) ---
FROM golang:1.25-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential gcc g++ pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH}

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# --- Runtime ---
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git wget \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -r -u 10001 -m -d /app -s /usr/sbin/nologin appuser

WORKDIR /app
COPY --from=builder --chown=appuser:appuser /out/server /app/server
COPY --from=builder --chown=appuser:appuser /out/migrate /app/migrate
COPY --chown=appuser:appuser migrations /app/migrations

RUN mkdir -p /app/workspace && chown -R appuser:appuser /app/workspace

USER appuser
ENV HTTP_ADDR=:8080 \
    MIGRATIONS_DIR=/app/migrations \
    WORKSPACE_ROOT=/app/workspace

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=25s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/server"]

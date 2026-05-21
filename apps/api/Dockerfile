# syntax=docker/dockerfile:1

FROM golang:1.24-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential gcc g++ pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -o /out/server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -r -u 10001 -m appuser
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY migrations /app/migrations
USER appuser
ENV HTTP_ADDR=:8080 MIGRATIONS_DIR=/app/migrations WORKSPACE_ROOT=/app/workspace
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/server"]

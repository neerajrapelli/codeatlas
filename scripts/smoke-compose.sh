#!/usr/bin/env bash
# Full-stack smoke: docker compose + Playwright against live API/nginx proxy.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

wait_url() {
  local url="$1"
  local max="${2:-90}"
  local i=0
  while [ "$i" -lt "$max" ]; do
    if curl -sf "$url" >/dev/null 2>&1; then
      echo "OK $url"
      return 0
    fi
    i=$((i + 1))
    sleep 2
  done
  echo "Timed out waiting for $url" >&2
  return 1
}

echo "Building and starting docker compose..."
docker compose up -d --build

wait_url "http://127.0.0.1:8080/health"
wait_url "http://127.0.0.1/api/health"

echo "Running Playwright (E2E_COMPOSE=1)..."
export E2E_COMPOSE=1
pnpm --filter @codeatlas/web exec playwright install chromium
pnpm --filter @codeatlas/web test:e2e

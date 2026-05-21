#!/usr/bin/env bash
# Run database migrations (use before scaling API replicas or via docker compose migrate service).
set -euo pipefail
cd "$(dirname "$0")/../apps/api"
exec go run ./cmd/migrate

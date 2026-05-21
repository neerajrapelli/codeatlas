.PHONY: help install dev dev-full build lint lint-ruff format typecheck clean docker-up docker-down docker-logs db-shell ai-sync index-repo test test-parser test-graph test-e2e smoke-compose smoke-compose-down

help:
	@echo "CodeAtlas development commands"
	@echo ""
	@echo "  make install     Install JS deps (pnpm) + sync Python tooling (optional)"
	@echo "  make dev         Run web + api only (skips Python AI stub)"
	@echo "  make dev-full    Run all dev tasks via Turborepo (includes apps/ai)"
	@echo "  make build       Build all packages/apps via Turborepo"
	@echo "  make lint        Lint via Turborepo"
	@echo "  make lint-ruff   Run Ruff on apps/ai (requires: make ai-sync)"
	@echo "  make format      Format repo with Prettier"
	@echo "  make typecheck   Typecheck TS + Go vet + Python compileall"
	@echo "  make clean       Clean build outputs (best-effort)"
	@echo "  make docker-up   Start local dependencies (Postgres + pgvector)"
	@echo "  make docker-down Stop local dependencies"
	@echo "  make docker-logs Tail Postgres logs"
	@echo "  make smoke-compose  Build stack + Playwright E2E against live API"
	@echo "  make smoke-compose-down  Stop full compose stack"
	@echo "  make db-shell    Open psql against local Postgres"
	@echo "  make ai-sync     Install Python dependencies for apps/ai (recommended)"
	@echo "  make index-repo  Ingest a local TypeScript repository (REPO=/path)"

install:
	pnpm install

dev:
	pnpm dev --filter @codeatlas/web --filter @codeatlas/api

dev-full:
	pnpm dev

test:
	cd apps/api && go test ./...

test-parser:
	cd apps/api && go test ./internal/indexer/...

test-graph:
	cd apps/api && go test ./internal/graphhierarchy/...

test-e2e:
	pnpm --filter @codeatlas/web test:e2e

smoke-compose:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-compose.ps1
else
	bash scripts/smoke-compose.sh
endif

smoke-compose-down:
	docker compose down

build:
	pnpm build

lint:
	pnpm lint

lint-ruff:
	pnpm --filter @codeatlas/ai lint:ruff

format:
	pnpm format

typecheck:
	pnpm typecheck

clean:
	pnpm clean

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f postgres

db-shell:
	docker compose exec postgres psql -U codeatlas -d codeatlas

ai-sync:
	python -m pip install -U pip
	python -m pip install -e "apps/ai[dev]"

index-repo:
	pnpm --filter @codeatlas/api index -- -repo "$(REPO)"

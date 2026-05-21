# Design Decisions

Each decision reflects **what is in the repository today**, with rationale inferred from structure and comments.

## ADR-1: Graph as source of truth

**Decision:** All intelligence (code structure, socio metrics, future signals) attaches to `repositories` / `files` rows—not standalone GitHub caches.

**Why:** Enables one query surface for map, inspector, and RAG; avoids sync drift between “GitHub features” and the architecture map.

**Evidence:** `file_metrics`, `commit_files`, `pr_files` FK to `files`; `GET /graph/clusters` returns `socioOverlay`.

---

## ADR-2: Go monolith API vs microservices

**Decision:** Primary backend is a single Go process (`cmd/server`) with packages under `internal/`.

**Why:** Simpler local dev, shared DB pool, synchronous indexer invocation, low operational overhead for MVP.

**Trade-off:** Horizontal scaling requires shared DB + stateless API replicas; long-running ingest jobs block only goroutines, not HTTP threads.

---

## ADR-3: TypeScript-first indexing (Tree-sitter)

**Decision:** MVP indexer uses `TreeSitterTypeScriptParser` and TS file scanner.

**Why:** Faster path to dependency and symbol extraction for typical web monorepos.

**Limitation:** Other languages are not parsed until new parsers/scanners are added.

**Files:** `internal/indexer/parser.go`, `scanner.go`.

---

## ADR-4: pgvector for semantic retrieval

**Decision:** Embeddings stored in `entity_embeddings`; retrieval uses cosine distance in SQL.

**Why:** Keeps semantic and structural graph in one database; avoids separate vector DB for MVP.

**Fallback:** Without `OPENAI_API_KEY`, `llm.NewLocalClient` provides stub embeddings (quality degraded—see FAQ).

---

## ADR-5: Provider manager with fallback chain

**Decision:** `internal/ai/providers/manager.go` registers multiple providers; chat/embed can fall back.

**Why:** Supports local dev without paid keys; allows swapping vendors via env.

**Reality:** Several providers return “not configured” errors until API keys and implementations are completed.

---

## ADR-6: SSE for streaming chat

**Decision:** `POST /ai/chat` with `"stream": true` returns `text/event-stream` events: `meta`, `token`, `done`, `error`.

**Why:** Better UX in `App.tsx` than waiting for full JSON body.

**File:** `internal/httpserver/server.go` (`writeSSE`).

---

## ADR-7: No application authentication (MVP)

**Decision:** No JWT/session middleware; only CORS restrictions for browsers.

**Why:** Faster iteration for local/single-tenant use.

**Risk:** Must place API behind VPN or API gateway auth before public exposure.

---

## ADR-8: GitHub socio sync as background Phase 1

**Decision:** `runSocioEnrichment` fires after code `ready`; does not block initial map.

**Why:** GitHub rate limits and commit detail fetches are slow; UX requires early `filesIndexed` graph.

**Caps:** `maxCommitDetail: 400`, paginated commit/PR lists in `ingestion/service.go`.

---

## ADR-9: BIGINT graph IDs, UUID socio IDs

**Decision:** Core graph entities use `BIGSERIAL`; socio entities use UUID (`pgcrypto`).

**Why:** Graph matches early migrations; socio tables added later with distributed-friendly IDs.

**Implication:** Joins always bridge `files.id` (BIGINT) ↔ socio tables.

---

## ADR-10: Vite `/api` proxy

**Decision:** Web dev server proxies `/api` → `http://localhost:8080` with path rewrite.

**Why:** Same-origin requests avoid CORS and localhost vs 127.0.0.1 mismatches.

**File:** `apps/web/vite.config.ts`, `apps/web/src/apiBase.ts`.

---

## ADR-11: Custom SQL migrations (not goose binary)

**Decision:** `internal/db/migrate.go` applies sorted `*.up.sql` files.

**Why:** Lightweight, no extra migration CLI dependency in CI (when added).

**Note:** `0002` files contain `-- +goose Up` comments but runner is custom.

---

## Future decisions (not implemented)

| Topic | Probable direction |
|-------|-------------------|
| Phase 2 signals | Batch LLM extraction, `confidence >= 0.7` filter |
| Phase 3 CI | Correlate `ci_runs` to `files` via commit SHA |
| Auth | API keys or OIDC at gateway |
| Multi-language | Per-language Tree-sitter parsers in indexer |

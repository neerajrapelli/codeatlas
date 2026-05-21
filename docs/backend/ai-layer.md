# AI Layer (Graph RAG)

## Location

All production AI chat logic runs **inside the Go API** (`internal/ai`), not the Python `apps/ai` service.

## Components

| File | Role |
|------|------|
| `service.go` | `PrepareChat`, `Answer`, `StreamCompletion` |
| `retriever.go` | Semantic seeds + dependency expansion + socio merge |
| `prompt.go` | System + user prompt assembly |
| `types.go` | `ChatRequest`, `ContextItem`, `ChatResponse` |
| `providers/manager.go` | Provider registration, fallback, embed/chat/stream |

## Chat request (HTTP)

```json
{
  "repositoryId": 1,
  "query": "What breaks if we change auth?",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "stream": true
}
```

| Field | Required | Notes |
|-------|----------|-------|
| `repositoryId` | Yes | Must have indexed files |
| `query` | Yes | User question |
| `provider` | No | Defaults to `AI_DEFAULT_PROVIDER` |
| `model` | No | Defaults to `AI_DEFAULT_MODEL` |
| `stream` | No | `true` → SSE |

## Retrieval pipeline

1. **Embed query** — `ProviderManager.Embed` (falls back across providers).
2. **Semantic seeds** — Top-K files from `entity_embeddings` (pgvector `<=>`).
3. **Expand** — BFS on `file_dependencies` (cap ~ `limit * 3` nodes).
4. **Load** — SQL aggregate imports/exports/symbols + dep counts.
5. **Socio merge** — `socio.Store.SocioContextForFiles` adds owner/risk/churn; boosts importance for hotspots.
6. **Trim** — Sort by importance; cap prompt via `AI_CONTEXT_TOKEN_BUDGET`.

## Prompt contract

System prompt (`prompt.go`):

- Answer only from provided context
- Explain impact and dependency paths
- Include **Related files** section with bullet paths

User prompt lines per file:

```
- file=path dep_out=N dep_in=N imports=[...] exports=[...] symbols=[...] owner=login bus_factor=N ...
```

## Streaming (SSE)

Events (JSON lines):

| type | Payload |
|------|---------|
| `meta` | `relatedFiles`, `provider`, `model` |
| `token` | `token` string delta |
| `done` | end |
| `error` | `error` message |

**Client:** `apps/web/src/App.tsx` `submitChat` consumes the stream.

## Providers

Registered in `cmd/server/main.go`:

| Provider | Env | Status |
|----------|-----|--------|
| `local` | — | Works (stub text/embed) |
| `openai` | `OPENAI_API_KEY` | Works when key set |
| `anthropic` | `ANTHROPIC_API_KEY` | Registered; implementation may error |
| `gemini` | `GEMINI_API_KEY` | Registered |
| `huggingface` | `HUGGINGFACE_API_KEY` | Registered |
| `openrouter` | `OPENROUTER_API_KEY` | Registered |

**UI note:** Web enables only `local` and `openai` in `ENABLED_PROVIDERS`.

## Related files in response

Derived from retrieved context items—used to highlight nodes on the map (`setHighlightedIds`).

## Python AI service

`apps/ai/src/codeatlas_ai/main.py` — FastAPI `GET /health` only.

**Assumption:** Future separation for GPU workloads or custom models; currently **disconnected** from `/ai/chat`.

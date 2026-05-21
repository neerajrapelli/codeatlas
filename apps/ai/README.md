# apps/ai — CodeAtlas AI Service (Python / FastAPI)

## Current Status: STUB — not required for local development

This service is intentionally minimal. It exists as a placeholder for future AI workloads
that are better suited to Python than Go.

## What it owns (future)

- Model fine-tuning pipelines
- Evaluation harnesses (parsing evals, retrieval evals, reasoning evals)
- Batch embedding generation jobs
- Experimental reasoning features

## What it does NOT own (these live in apps/api)

- Real-time AI chat → apps/api/internal/ai
- Provider abstraction (OpenAI, Anthropic, Gemini) → apps/api/internal/ai/providers
- Graph-augmented retrieval → apps/api/internal/ai

## Running it

Not required for local dev. Only start it if you're working on evaluation features.

```bash
make ai-sync && pnpm dev --filter @codeatlas/ai
```

## Health check

`GET http://localhost:8001/health` → `{ "status": "ok" }`

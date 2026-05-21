# Performance and Scaling

## Implemented optimizations

| Area | Technique | Location |
|------|-----------|----------|
| Graph map | Clustered layers per prefix (not full repo at once) | `graphhierarchy` |
| Layout | ELK layered layout client-side | `HierarchyGraph.tsx` |
| Retrieval | Top-K semantic seeds + capped BFS expansion | `ai/retriever.go` |
| Prompt | Token budget truncation | `ai/prompt.go` |
| Socio ingest | Paginated GitHub lists; commit detail budget (400) | `ingestion/service.go` |
| DB | Indexes on `repository_id`, `file_id`, hotspots | migrations |
| Partial UX | Map visible when `filesIndexed > 0` | `App.tsx` |

## Known bottlenecks

| Bottleneck | Impact | Mitigation today |
|------------|--------|------------------|
| Full reindex | O(files) parse + embed | Background job; user waits on progress |
| `GET /graph/clusters` | DB aggregation per prefix | 60s handler timeout |
| GitHub API | Rate limits | Backoff in `github/client.go` |
| ELK layout | CPU on large layers | Only current prefix loaded |
| No query cache | Repeated cluster fetches | — |

## Scaling strategies (recommended)

### API tier

- Run **N stateless replicas** behind load balancer
- Move `WORKSPACE_ROOT` to **shared filesystem** or refactor ingest to object storage + workers
- Separate **worker process** for ingest/index (queue: SQS, Redis, or DB job table)

### Database

- **pgvector** index (IVFFlat/HNSW) when embeddings > 100k rows
- Partition `commits` / `commit_files` by `repository_id` for large orgs
- Archive old socio data with retention policy

### Frontend

- CDN for `dist/` static assets
- Debounce cluster fetches on rapid breadcrumb navigation
- Virtualize file lists if prefix contains thousands of files *(not implemented)*

### AI

- Cache embeddings per file hash + commit SHA
- Batch socio LLM extraction (Phase 2)
- Route chat to cheaper models for summarization steps

## Performance testing (manual)

1. Index a monorepo (e.g. 5k TS files)—record `stage_metadata.indexDurationMs`
2. Open map at root vs deep prefix—compare `GET /graph/clusters` latency
3. Chat with narrow vs broad queries—measure time-to-first SSE token

## Future improvements

See [../onboarding/faq.md](../onboarding/faq.md) and product Phase 2/3 schema for engineering memory and CI correlation.

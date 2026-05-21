# Socio-Technical Ingestion

## Purpose

Enrich the code graph with **organizational signals**: who changes files, how often, PR activity, and derived **risk metrics** (hotspots, bus factor).

**Package:** `internal/ingestion`  
**Store/query:** `internal/socio`  
**GitHub API:** `internal/github`

## Phase 1 (implemented)

`ingestion.Service.RunPhase1GitHubHistory(ctx, repositoryID)`

### Preconditions

| Condition | Result |
|-----------|--------|
| `repositories.source_type != github` | Run `skipped` |
| `GITHUB_TOKEN` empty | Run `skipped` |
| Unparseable `source_url` | Run `failed` |

### Steps (recorded in `socio_ingestion_steps`)

1. `resolve_repository` — build path → `file_id` index
2. `sync_commits` — paginated commit list + bounded commit detail fetches
3. `sync_pull_requests` — PR list + per-PR files
4. `compute_file_metrics` — aggregate 90-day window, replace metrics tables

### Tables written

- `contributors`, `commits`, `commit_files`
- `pull_requests`, `pr_files`
- `file_metrics`, `contributor_file_ownership`
- `socio_ingestion_runs`, `socio_ingestion_steps`

### Metrics algorithm

` socio.ComputeFileMetrics` (`internal/socio/metrics.go`):

- Window: **90 days** of `commit_files` + `pr_files`
- **Churn:** additions + deletions
- **Hotspot score:** churn vs p90 normalization (≥ 0.75 → hotspot)
- **Bus factor risk:** ≤ 1 author with ≥ 3 commits
- **Risk level:** `low` | `medium` | `high` | `critical` composite

### GitHub client behavior

`internal/github/client.go`:

- Base URL: `https://api.github.com`
- Retries with exponential backoff + jitter (`maxRetries: 5`)
- Rate-limit aware (reads `X-RateLimit-*` headers)
- **Token:** `GITHUB_TOKEN` env (PAT or app installation token string)

**Not stored:** Raw GitHub JSON blobs—only normalized rows.

## Read APIs

| Endpoint | Service method |
|----------|----------------|
| `GET .../ownership` | `QueryService.GetOwnership` |
| `GET .../hotspots` | `QueryService.GetHotspots` |
| `GET .../ingestion/status` | `QueryService.BuildIngestionStatus` |
| `socioOverlay` on clusters | `QueryService.GetFileOverlays` |

## Phase 2 (schema only)

Tables exist in `0006_socio_technical.up.sql`:

- `pr_reviews`, `pr_comments`, `issues`, `issue_file_refs`, `architecture_signals`

**Signal types (check constraint):**

`technical_debt`, `coupling_warning`, `migration_intent`, `known_fragility`, `ownership_boundary`, `architectural_decision`

**Planned rules (from product spec, not fully coded):**

- AI extraction via existing provider manager
- Persist only `confidence > 0.7`
- Batch comments for cost control

## Phase 3 (schema only)

- `ci_runs` for workflow correlation
- Planned API: `GET /repositories/{id}/risk-summary` *(not in httpserver yet)*

## Trigger points

```go
// internal/repoingest/service.go
func (s *Service) runSocioEnrichment(repositoryID int64) {
    go func() {
        s.socioIngest.RunPhase1GitHubHistory(ctx, repositoryID)
    }()
}
```

Called after: sync ingest ready, background ingest ready, reindex ready.

## Observability

Structured logs:

- `socio_ingestion_skipped`
- `socio_ingestion_failed`
- `socio_phase1_complete`

DB: query `socio_ingestion_runs` / `socio_ingestion_steps` for audit trail.

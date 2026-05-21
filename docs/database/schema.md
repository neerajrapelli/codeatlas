# Database Schema

## Engine

- **PostgreSQL 16** with extensions:
  - `vector` (pgvector) — `entity_embeddings`
  - `pgcrypto` — UUID defaults for socio tables

**Local image:** `pgvector/pgvector:pg16` (`docker-compose.yml`)

**Migrations:** `apps/api/migrations/*.up.sql`  
**Tracker table:** `schema_migrations` (custom runner in `internal/db/migrate.go`)

## Migration history

| Version | File | Summary |
|---------|------|---------|
| 0002 | `0002_indexing_graph.up.sql` | Core graph tables |
| 0003 | `0003_ai_embeddings.up.sql` | `entity_embeddings` + vector |
| 0004 | `0004_repository_sources.up.sql` | Source metadata on `repositories` |
| 0005 | `0005_ingestion_progress.up.sql` | Progress columns + stage metadata |
| 0006 | `0006_socio_technical.up.sql` | Socio + Phase 2/3 tables |

*Note: No `0001` migration in repository.*

## Core graph entities

### `repositories`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGSERIAL | PK |
| `name` | TEXT | Display name |
| `root_path` | TEXT UNIQUE | Workspace path key |
| `source_type` | TEXT | github, gitlab, bitbucket, zip |
| `source_url` | TEXT | Remote URL or empty for zip |
| `branch` | TEXT | |
| `workspace_path` | TEXT | Absolute path on API host |
| `status` | TEXT | queued … ready, failed |
| `progress_percent` | FLOAT | 0–100 |
| `files_indexed`, `symbols_indexed`, … | INT | Metrics |
| `current_stage` | TEXT | Indexing sub-stage |
| `stage_metadata` | JSONB | Timing/debug |
| `error_details` | TEXT | Failure message |

### `files`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGSERIAL | PK |
| `repository_id` | BIGINT FK | CASCADE |
| `relative_path` | TEXT | Unique per repo |

### `symbols`

AST-derived symbols with `kind`, line/column span, `exported` flag.

### `file_imports` / `file_exports`

Import module paths and export names per file.

### `file_dependencies`

Directed edges `from_file_id` → `to_file_id` (resolved internal deps).

### `entity_embeddings`

| Column | Purpose |
|--------|---------|
| `repository_id` | Scope |
| `file_id` / entity refs | Embedded node |
| `embedding` | `vector` type |

## Socio-technical entities (Phase 1)

### `contributors`

GitHub users per repository (`external_id`, `login`).

### `commits` / `commit_files`

Commit SHAs linked to `files` with `change_kind` and line churn.

### `pull_requests` / `pr_files`

PR metadata and per-file diffs.

### `file_metrics`

Aggregated per-file (90d):

- `churn_score`, `commit_count_90d`, `unique_authors_90d`
- `bus_factor`, `hotspot_score`, `risk_level`
- `is_hotspot`, `has_bus_factor_risk`
- `dominant_owner_id`, `dominant_owner_share`

### `contributor_file_ownership`

Many-to-many ownership shares per file.

### `socio_ingestion_runs` / `socio_ingestion_steps`

Job observability: phase, status, percent, per-step duration and counts.

## Phase 2 tables (schema only)

- `pr_reviews`, `pr_comments`
- `issues`, `issue_file_refs`
- `architecture_signals` — AI-extracted normalized signals

## Phase 3 tables (schema only)

- `ci_runs` — workflow runs keyed by `external_id`

## Graph relationship semantics (logical)

These are **not** separate edge tables; they are implied relations:

| Logical edge | Storage |
|--------------|---------|
| `file` imports `file` | `file_dependencies` |
| `commit` modified `file` | `commit_files` |
| `pr` changed `file` | `pr_files` |
| `contributor` owns `file` | `contributor_file_ownership` + `file_metrics.dominant_owner_id` |
| `file` is hotspot | `file_metrics.is_hotspot` |

## Indexes (high level)

- Repository-scoped indexes on foreign keys (`repository_id`, `file_id`)
- Hotspot/risk partial indexes on `file_metrics`
- Time-ordered indexes on `commits`, `pull_requests`, `ci_runs`

## Deletion behavior

`DELETE /repositories/{id}` removes repository row → **ON DELETE CASCADE** clears all dependent graph and socio data.

See [erd.md](./erd.md) for diagram.

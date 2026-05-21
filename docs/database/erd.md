# Entity Relationship Diagram

## Core code graph

```mermaid
erDiagram
  repositories ||--o{ files : contains
  files ||--o{ symbols : defines
  files ||--o{ file_imports : imports
  files ||--o{ file_exports : exports
  files ||--o{ file_dependencies : from
  files ||--o{ file_dependencies : to
  repositories ||--o{ entity_embeddings : scopes
  files ||--o{ entity_embeddings : embeds
```

## Socio-technical (Phase 1)

```mermaid
erDiagram
  repositories ||--o{ contributors : has
  repositories ||--o{ commits : has
  contributors ||--o{ commits : authors
  commits ||--o{ commit_files : touches
  files ||--o{ commit_files : touched_by
  repositories ||--o{ pull_requests : has
  pull_requests ||--o{ pr_files : touches
  files ||--o{ pr_files : touched_by
  repositories ||--o{ file_metrics : metrics
  files ||--o{ file_metrics : measured
  contributors ||--o{ file_metrics : dominant_owner
  files ||--o{ contributor_file_ownership : ownership
  contributors ||--o{ contributor_file_ownership : share
  repositories ||--o{ socio_ingestion_runs : jobs
  socio_ingestion_runs ||--o{ socio_ingestion_steps : steps
```

## Phase 2/3 (tables present)

```mermaid
erDiagram
  pull_requests ||--o{ pr_comments : has
  pull_requests ||--o{ pr_reviews : has
  repositories ||--o{ issues : has
  issues ||--o{ issue_file_refs : mentions
  files ||--o{ issue_file_refs : referenced
  repositories ||--o{ architecture_signals : signals
  files ||--o{ architecture_signals : optional
  repositories ||--o{ ci_runs : runs
```

## ID types

| Domain | PK type |
|--------|---------|
| repositories, files, symbols | BIGINT |
| contributors, commits, PRs, runs | UUID |

Joins always use `files.id` (BIGINT) when linking socio data to the code graph.

-- Socio-technical intelligence layer (graph enrichment). Phase 1 core + schema for phases 2–3.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS contributors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  external_id TEXT NOT NULL,
  login TEXT NOT NULL,
  display_name TEXT,
  avatar_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (repository_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_contributors_repo ON contributors(repository_id);
CREATE INDEX IF NOT EXISTS idx_contributors_login ON contributors(repository_id, login);

CREATE TABLE IF NOT EXISTS commits (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  sha TEXT NOT NULL,
  author_contributor_id UUID REFERENCES contributors(id) ON DELETE SET NULL,
  committed_at TIMESTAMPTZ NOT NULL,
  message_preview TEXT NOT NULL DEFAULT '',
  additions INTEGER NOT NULL DEFAULT 0,
  deletions INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (repository_id, sha)
);

CREATE INDEX IF NOT EXISTS idx_commits_repo_time ON commits(repository_id, committed_at DESC);

CREATE TABLE IF NOT EXISTS commit_files (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  commit_id UUID NOT NULL REFERENCES commits(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  change_kind TEXT NOT NULL CHECK (change_kind IN ('add', 'modify', 'delete', 'rename')),
  additions INTEGER NOT NULL DEFAULT 0,
  deletions INTEGER NOT NULL DEFAULT 0,
  UNIQUE (commit_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_commit_files_file ON commit_files(file_id);

CREATE TABLE IF NOT EXISTS pull_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  external_number INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'open',
  author_contributor_id UUID REFERENCES contributors(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL,
  merged_at TIMESTAMPTZ,
  closed_at TIMESTAMPTZ,
  additions INTEGER NOT NULL DEFAULT 0,
  deletions INTEGER NOT NULL DEFAULT 0,
  changed_files INTEGER NOT NULL DEFAULT 0,
  created_at_ingested TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (repository_id, external_number)
);

CREATE INDEX IF NOT EXISTS idx_pull_requests_repo ON pull_requests(repository_id, created_at DESC);

CREATE TABLE IF NOT EXISTS pr_files (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  change_kind TEXT NOT NULL CHECK (change_kind IN ('add', 'modify', 'delete', 'rename')),
  additions INTEGER NOT NULL DEFAULT 0,
  deletions INTEGER NOT NULL DEFAULT 0,
  UNIQUE (pull_request_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_files_file ON pr_files(file_id);

CREATE TABLE IF NOT EXISTS file_metrics (
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  churn_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  commit_count_90d INTEGER NOT NULL DEFAULT 0,
  unique_authors_90d INTEGER NOT NULL DEFAULT 0,
  bus_factor INTEGER NOT NULL DEFAULT 0,
  hotspot_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  risk_level TEXT NOT NULL DEFAULT 'low' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
  is_hotspot BOOLEAN NOT NULL DEFAULT FALSE,
  has_bus_factor_risk BOOLEAN NOT NULL DEFAULT FALSE,
  dominant_owner_id UUID REFERENCES contributors(id) ON DELETE SET NULL,
  dominant_owner_share DOUBLE PRECISION NOT NULL DEFAULT 0,
  last_activity_at TIMESTAMPTZ,
  computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (repository_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_file_metrics_hotspot ON file_metrics(repository_id, is_hotspot) WHERE is_hotspot;
CREATE INDEX IF NOT EXISTS idx_file_metrics_risk ON file_metrics(repository_id, risk_level);

CREATE TABLE IF NOT EXISTS contributor_file_ownership (
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  contributor_id UUID NOT NULL REFERENCES contributors(id) ON DELETE CASCADE,
  commit_count INTEGER NOT NULL DEFAULT 0,
  ownership_share DOUBLE PRECISION NOT NULL DEFAULT 0,
  PRIMARY KEY (repository_id, file_id, contributor_id)
);

CREATE INDEX IF NOT EXISTS idx_contributor_file_ownership_file ON contributor_file_ownership(file_id);

-- Ingestion observability (socio-technical phases)
CREATE TABLE IF NOT EXISTS socio_ingestion_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  phase TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped', 'partial')),
  completion_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  error_details TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_socio_ingestion_runs_repo ON socio_ingestion_runs(repository_id, created_at DESC);

CREATE TABLE IF NOT EXISTS socio_ingestion_steps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES socio_ingestion_runs(id) ON DELETE CASCADE,
  step TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
  duration_ms BIGINT,
  items_processed INTEGER NOT NULL DEFAULT 0,
  failure_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_socio_ingestion_steps_run ON socio_ingestion_steps(run_id);

-- Phase 2 schema (ingestion wired in phase 2)
CREATE TABLE IF NOT EXISTS pr_reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
  reviewer_contributor_id UUID REFERENCES contributors(id) ON DELETE SET NULL,
  state TEXT NOT NULL DEFAULT '',
  submitted_at TIMESTAMPTZ,
  UNIQUE (pull_request_id, reviewer_contributor_id, submitted_at)
);

CREATE TABLE IF NOT EXISTS pr_comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
  author_contributor_id UUID REFERENCES contributors(id) ON DELETE SET NULL,
  body_preview TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  external_id TEXT NOT NULL,
  UNIQUE (pull_request_id, external_id)
);

CREATE TABLE IF NOT EXISTS issues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  external_number INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'open',
  author_contributor_id UUID REFERENCES contributors(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL,
  closed_at TIMESTAMPTZ,
  UNIQUE (repository_id, external_number)
);

CREATE TABLE IF NOT EXISTS issue_file_refs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  ref_kind TEXT NOT NULL DEFAULT 'mentioned',
  UNIQUE (issue_id, file_id)
);

CREATE TABLE IF NOT EXISTS architecture_signals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  file_id BIGINT REFERENCES files(id) ON DELETE CASCADE,
  signal_type TEXT NOT NULL CHECK (signal_type IN (
    'technical_debt', 'coupling_warning', 'migration_intent',
    'known_fragility', 'ownership_boundary', 'architectural_decision'
  )),
  summary TEXT NOT NULL,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
  source_kind TEXT NOT NULL,
  source_id UUID,
  extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_architecture_signals_repo ON architecture_signals(repository_id, signal_type);

-- Phase 3 schema
CREATE TABLE IF NOT EXISTS ci_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  workflow_name TEXT NOT NULL DEFAULT '',
  run_number BIGINT NOT NULL DEFAULT 0,
  conclusion TEXT,
  status TEXT NOT NULL DEFAULT '',
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  head_sha TEXT,
  external_id TEXT NOT NULL,
  UNIQUE (repository_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_ci_runs_repo ON ci_runs(repository_id, started_at DESC);

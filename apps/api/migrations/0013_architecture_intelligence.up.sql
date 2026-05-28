-- +goose Up
CREATE TABLE IF NOT EXISTS architecture_decisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'proposed',
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.0,
  tradeoffs JSONB NOT NULL DEFAULT '[]'::jsonb,
  affected_modules JSONB NOT NULL DEFAULT '[]'::jsonb,
  affected_files JSONB NOT NULL DEFAULT '[]'::jsonb,
  participants JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_kind TEXT NOT NULL DEFAULT '',
  source_ref TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_arch_decision_status CHECK (status IN ('proposed', 'accepted', 'rejected', 'deprecated'))
);

CREATE INDEX IF NOT EXISTS idx_arch_decisions_repo_tenant
  ON architecture_decisions(tenant_id, repository_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_arch_decisions_status
  ON architecture_decisions(repository_id, status);

CREATE TABLE IF NOT EXISTS architecture_decision_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  decision_id UUID NOT NULL REFERENCES architecture_decisions(id) ON DELETE CASCADE,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  link_kind TEXT NOT NULL,
  module_path TEXT NOT NULL DEFAULT '',
  file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,
  symbol_id BIGINT REFERENCES symbols(id) ON DELETE SET NULL,
  pull_request_id UUID REFERENCES pull_requests(id) ON DELETE SET NULL,
  issue_id UUID REFERENCES issues(id) ON DELETE SET NULL,
  commit_id UUID REFERENCES commits(id) ON DELETE SET NULL,
  source_url TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_arch_decision_links_decision ON architecture_decision_links(decision_id);
CREATE INDEX IF NOT EXISTS idx_arch_decision_links_repo_tenant_kind
  ON architecture_decision_links(tenant_id, repository_id, link_kind);
CREATE INDEX IF NOT EXISTS idx_arch_decision_links_module
  ON architecture_decision_links(repository_id, module_path);

CREATE TABLE IF NOT EXISTS architecture_decision_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  decision_id UUID NOT NULL REFERENCES architecture_decisions(id) ON DELETE CASCADE,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  event_type TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  actor_login TEXT NOT NULL DEFAULT '',
  event_at TIMESTAMPTZ NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_arch_decision_events_repo_tenant_event
  ON architecture_decision_events(tenant_id, repository_id, event_at DESC);
CREATE INDEX IF NOT EXISTS idx_arch_decision_events_decision
  ON architecture_decision_events(decision_id, event_at DESC);

CREATE TABLE IF NOT EXISTS discussion_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  source_kind TEXT NOT NULL,
  source_id TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  author_login TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  module_hints JSONB NOT NULL DEFAULT '[]'::jsonb,
  participant_hints JSONB NOT NULL DEFAULT '[]'::jsonb,
  occurred_at TIMESTAMPTZ,
  search_document TSVECTOR,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_discussion_docs_unique
  ON discussion_documents(repository_id, tenant_id, source_kind, source_id);
CREATE INDEX IF NOT EXISTS idx_discussion_docs_repo_tenant_time
  ON discussion_documents(tenant_id, repository_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_discussion_docs_search_gin
  ON discussion_documents USING GIN(search_document);

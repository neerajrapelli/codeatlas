-- Denormalize tenant_id onto repository-scoped tables for defense-in-depth isolation.

-- Graph / indexing
ALTER TABLE files ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE files f SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = f.repository_id;
CREATE INDEX IF NOT EXISTS idx_files_tenant_repo ON files (tenant_id, repository_id);

ALTER TABLE symbols ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE symbols s SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = s.repository_id;
CREATE INDEX IF NOT EXISTS idx_symbols_tenant_repo ON symbols (tenant_id, repository_id);

ALTER TABLE file_imports ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE file_imports t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE file_exports ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE file_exports t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE file_dependencies ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE file_dependencies t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE entity_embeddings ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE entity_embeddings t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

-- Architecture rules
ALTER TABLE architecture_rules ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE architecture_rules t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE rule_violations ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE rule_violations t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

-- Socio-technical (Phase 1–2)
ALTER TABLE contributors ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE contributors t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE commits ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE commits t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE pull_requests ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE pull_requests t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE file_metrics ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE file_metrics t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE contributor_file_ownership ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE contributor_file_ownership t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE architecture_signals ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE architecture_signals t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE socio_ingestion_runs ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE socio_ingestion_runs t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE issues ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE issues t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

-- Teams & jobs
ALTER TABLE teams ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE teams t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE team_file_ownership ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE team_file_ownership t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

ALTER TABLE ingestion_jobs ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
UPDATE ingestion_jobs t SET tenant_id = r.tenant_id FROM repositories r WHERE r.id = t.repository_id;

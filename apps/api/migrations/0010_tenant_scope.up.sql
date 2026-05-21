-- Scope repositories to JWT tenant_id for multi-tenant isolation.
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';

UPDATE repositories SET tenant_id = 'default' WHERE tenant_id = '' OR tenant_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_repositories_tenant ON repositories (tenant_id);

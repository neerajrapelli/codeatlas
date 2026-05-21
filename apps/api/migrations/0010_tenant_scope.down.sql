DROP INDEX IF EXISTS idx_repositories_tenant;
ALTER TABLE repositories DROP COLUMN IF EXISTS tenant_id;

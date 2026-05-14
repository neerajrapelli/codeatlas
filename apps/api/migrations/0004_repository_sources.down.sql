DROP INDEX IF EXISTS idx_repositories_created_at;
DROP INDEX IF EXISTS idx_repositories_status;

ALTER TABLE repositories DROP CONSTRAINT IF EXISTS repositories_status_check;

ALTER TABLE repositories
  DROP COLUMN IF EXISTS indexed_at,
  DROP COLUMN IF EXISTS indexing_started_at,
  DROP COLUMN IF EXISTS error_details,
  DROP COLUMN IF EXISTS status,
  DROP COLUMN IF EXISTS workspace_path,
  DROP COLUMN IF EXISTS branch,
  DROP COLUMN IF EXISTS source_url,
  DROP COLUMN IF EXISTS source_type;

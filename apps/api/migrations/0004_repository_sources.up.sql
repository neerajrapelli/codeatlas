ALTER TABLE repositories
  ADD COLUMN IF NOT EXISTS source_type TEXT,
  ADD COLUMN IF NOT EXISTS source_url TEXT,
  ADD COLUMN IF NOT EXISTS branch TEXT,
  ADD COLUMN IF NOT EXISTS workspace_path TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'queued',
  ADD COLUMN IF NOT EXISTS error_details TEXT,
  ADD COLUMN IF NOT EXISTS indexing_started_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS indexed_at TIMESTAMPTZ;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'repositories_status_check'
  ) THEN
    ALTER TABLE repositories
      ADD CONSTRAINT repositories_status_check CHECK (
        status IN ('queued','cloning','extracting','indexing','ready','failed')
      );
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_repositories_status ON repositories(status);
CREATE INDEX IF NOT EXISTS idx_repositories_created_at ON repositories(created_at DESC);

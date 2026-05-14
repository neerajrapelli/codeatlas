ALTER TABLE repositories
  ADD COLUMN IF NOT EXISTS progress_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS files_indexed INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS symbols_indexed INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS edges_indexed INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS embeddings_indexed INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_stage TEXT NOT NULL DEFAULT 'queued',
  ADD COLUMN IF NOT EXISTS stage_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ADD COLUMN IF NOT EXISTS clone_duration_ms BIGINT,
  ADD COLUMN IF NOT EXISTS extract_duration_ms BIGINT,
  ADD COLUMN IF NOT EXISTS parse_duration_ms BIGINT,
  ADD COLUMN IF NOT EXISTS graph_duration_ms BIGINT,
  ADD COLUMN IF NOT EXISTS embedding_duration_ms BIGINT,
  ADD COLUMN IF NOT EXISTS total_duration_ms BIGINT;

ALTER TABLE repositories DROP CONSTRAINT IF EXISTS repositories_status_check;

UPDATE repositories
SET status = CASE
  WHEN status = 'indexing' THEN 'building_graph'
  WHEN status IN ('queued','cloning','extracting','parsing','building_graph','generating_embeddings','ready','failed') THEN status
  ELSE 'failed'
END,
current_stage = CASE
  WHEN status = 'indexing' THEN 'building_graph'
  WHEN status IN ('queued','cloning','extracting','parsing','building_graph','generating_embeddings','ready','failed') THEN status
  ELSE 'failed'
END;

ALTER TABLE repositories
  ADD CONSTRAINT repositories_status_check CHECK (
    status IN ('queued','cloning','extracting','parsing','building_graph','generating_embeddings','ready','failed')
  );

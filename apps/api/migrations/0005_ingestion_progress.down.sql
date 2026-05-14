ALTER TABLE repositories DROP CONSTRAINT IF EXISTS repositories_status_check;
ALTER TABLE repositories
  ADD CONSTRAINT repositories_status_check CHECK (
    status IN ('queued','cloning','extracting','indexing','ready','failed')
  );

ALTER TABLE repositories
  DROP COLUMN IF EXISTS total_duration_ms,
  DROP COLUMN IF EXISTS embedding_duration_ms,
  DROP COLUMN IF EXISTS graph_duration_ms,
  DROP COLUMN IF EXISTS parse_duration_ms,
  DROP COLUMN IF EXISTS extract_duration_ms,
  DROP COLUMN IF EXISTS clone_duration_ms,
  DROP COLUMN IF EXISTS last_heartbeat,
  DROP COLUMN IF EXISTS stage_metadata,
  DROP COLUMN IF EXISTS current_stage,
  DROP COLUMN IF EXISTS embeddings_indexed,
  DROP COLUMN IF EXISTS edges_indexed,
  DROP COLUMN IF EXISTS symbols_indexed,
  DROP COLUMN IF EXISTS files_indexed,
  DROP COLUMN IF EXISTS progress_percent;

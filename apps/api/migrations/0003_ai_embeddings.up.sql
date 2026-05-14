CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS entity_embeddings (
  id BIGSERIAL PRIMARY KEY,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL CHECK (entity_type IN ('file', 'symbol', 'import')),
  file_id BIGINT REFERENCES files(id) ON DELETE CASCADE,
  symbol_id BIGINT REFERENCES symbols(id) ON DELETE CASCADE,
  import_id BIGINT REFERENCES file_imports(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  embedding vector(1536) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entity_embeddings_repo_type ON entity_embeddings(repository_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_entity_embeddings_file ON entity_embeddings(file_id);
CREATE INDEX IF NOT EXISTS idx_entity_embeddings_symbol ON entity_embeddings(symbol_id);

CREATE INDEX IF NOT EXISTS idx_entity_embeddings_vector
  ON entity_embeddings
  USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

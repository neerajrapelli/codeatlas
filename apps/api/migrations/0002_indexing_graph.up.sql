-- +goose Up
CREATE TABLE IF NOT EXISTS repositories (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  root_path TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS files (
  id BIGSERIAL PRIMARY KEY,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  relative_path TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(repository_id, relative_path)
);

CREATE TABLE IF NOT EXISTS symbols (
  id BIGSERIAL PRIMARY KEY,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  exported BOOLEAN NOT NULL DEFAULT FALSE,
  start_line INTEGER NOT NULL,
  start_col INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  end_col INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS file_imports (
  id BIGSERIAL PRIMARY KEY,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  module_path TEXT NOT NULL,
  is_type_only BOOLEAN NOT NULL DEFAULT FALSE,
  is_external BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS file_exports (
  id BIGSERIAL PRIMARY KEY,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  export_name TEXT NOT NULL,
  source_path TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS file_dependencies (
  id BIGSERIAL PRIMARY KEY,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  from_file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  to_file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(repository_id, from_file_id, to_file_id)
);

CREATE INDEX IF NOT EXISTS idx_files_repository ON files(repository_id);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_imports_file ON file_imports(file_id);
CREATE INDEX IF NOT EXISTS idx_deps_from_file ON file_dependencies(from_file_id);

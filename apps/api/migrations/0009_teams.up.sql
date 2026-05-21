CREATE TABLE IF NOT EXISTS teams (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  slug            TEXT NOT NULL,
  display_name    TEXT NOT NULL,
  color           TEXT NOT NULL DEFAULT '#4a9eff',
  source          TEXT NOT NULL DEFAULT 'codeowners',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (repository_id, slug)
);

CREATE TABLE IF NOT EXISTS team_file_ownership (
  repository_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  file_id         BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  ownership_share DOUBLE PRECISION NOT NULL DEFAULT 1,
  PRIMARY KEY (repository_id, team_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_team_file_ownership_file ON team_file_ownership(file_id);

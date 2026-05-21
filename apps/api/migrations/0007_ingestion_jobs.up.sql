-- Postgres-backed ingestion job queue

DO $$ BEGIN
  CREATE TYPE job_status AS ENUM ('queued', 'running', 'complete', 'failed');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS ingestion_jobs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  phase           INT NOT NULL DEFAULT 1,
  status          job_status NOT NULL DEFAULT 'queued',
  current_step    TEXT,
  progress_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_msg       TEXT,
  metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
  queued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at      TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_repo_status ON ingestion_jobs(repository_id, status);
CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_queued ON ingestion_jobs(status, queued_at) WHERE status = 'queued';

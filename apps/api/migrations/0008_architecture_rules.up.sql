CREATE TABLE IF NOT EXISTS architecture_rules (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,
  description     TEXT,
  rule_type       TEXT NOT NULL CHECK (rule_type IN (
    'no_import',
    'must_import',
    'layer_order',
    'no_circular'
  )),
  source_pattern  TEXT NOT NULL,
  target_pattern  TEXT NOT NULL,
  severity        TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('error', 'warning', 'info')),
  enabled         BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_architecture_rules_repo ON architecture_rules(repository_id);

CREATE TABLE IF NOT EXISTS rule_violations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id         UUID NOT NULL REFERENCES architecture_rules(id) ON DELETE CASCADE,
  repository_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  source_file     TEXT NOT NULL,
  target_file     TEXT NOT NULL,
  detected_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at     TIMESTAMPTZ,
  is_active       BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_rule_violations_active ON rule_violations(repository_id, is_active) WHERE is_active = true;

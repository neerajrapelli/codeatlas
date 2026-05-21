-- Multi-provider VCS auth and repo source linkage

CREATE TABLE IF NOT EXISTS provider_tokens (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       TEXT NOT NULL,
  user_subject    TEXT NOT NULL,
  provider        TEXT NOT NULL CHECK (provider IN ('github','gitlab','bitbucket')),
  token_type      TEXT NOT NULL DEFAULT 'oauth' CHECK (token_type IN ('oauth','pat','app_password')),
  encrypted_token BYTEA NOT NULL,
  scopes          TEXT[] NOT NULL DEFAULT '{}',
  external_user_id TEXT,
  expires_at      TIMESTAMPTZ,
  metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, user_subject, provider)
);

CREATE INDEX IF NOT EXISTS idx_provider_tokens_tenant ON provider_tokens(tenant_id, provider);

CREATE TABLE IF NOT EXISTS repo_sources (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repository_id        BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  provider             TEXT NOT NULL CHECK (provider IN ('github','gitlab','bitbucket','zip')),
  external_repo_id     TEXT,
  external_repo_full_name TEXT,
  provider_token_id    UUID REFERENCES provider_tokens(id) ON DELETE SET NULL,
  access_metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (repository_id)
);

CREATE INDEX IF NOT EXISTS idx_repo_sources_provider ON repo_sources(provider, external_repo_id);

CREATE TABLE IF NOT EXISTS oauth_states (
  state         TEXT PRIMARY KEY,
  tenant_id     TEXT NOT NULL,
  user_subject  TEXT NOT NULL,
  provider      TEXT NOT NULL,
  redirect_uri  TEXT NOT NULL,
  expires_at    TIMESTAMPTZ NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);

package vcsauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool   *pgxpool.Pool
	cipher *Cipher
}

func NewStore(pool *pgxpool.Pool, cipher *Cipher) *Store {
	return &Store{pool: pool, cipher: cipher}
}

func (s *Store) UpsertToken(ctx context.Context, t ProviderToken, plaintext string) (uuid.UUID, error) {
	if s.cipher == nil {
		return uuid.Nil, errors.New("token encryption not configured")
	}
	enc, err := s.cipher.Encrypt([]byte(plaintext))
	if err != nil {
		return uuid.Nil, err
	}
	id := t.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	meta, _ := json.Marshal(t.Metadata)
	var expires *time.Time
	if t.ExpiresAt != nil {
		expires = t.ExpiresAt
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO provider_tokens (id, tenant_id, user_subject, provider, token_type, encrypted_token, scopes, external_user_id, expires_at, metadata, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb, now())
ON CONFLICT (tenant_id, user_subject, provider) DO UPDATE SET
  token_type = EXCLUDED.token_type,
  encrypted_token = EXCLUDED.encrypted_token,
  scopes = EXCLUDED.scopes,
  external_user_id = EXCLUDED.external_user_id,
  expires_at = EXCLUDED.expires_at,
  metadata = EXCLUDED.metadata,
  updated_at = now()
`, id, t.TenantID, t.UserSubject, string(t.Provider), string(t.TokenType), enc, t.Scopes, nullStr(t.ExternalUserID), expires, meta)
	return id, err
}

func (s *Store) GetPlaintextToken(ctx context.Context, tenantID, userSubject string, provider Provider) (string, *ProviderToken, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, tenant_id, user_subject, provider, token_type, encrypted_token, scopes, external_user_id, expires_at, metadata, created_at, updated_at
FROM provider_tokens
WHERE tenant_id = $1 AND user_subject = $2 AND provider = $3
`, tenantID, userSubject, string(provider))
	var pt ProviderToken
	var enc []byte
	var prov, tt string
	var meta []byte
	var extUser *string
	err := row.Scan(&pt.ID, &pt.TenantID, &pt.UserSubject, &prov, &tt, &enc, &pt.Scopes, &extUser, &pt.ExpiresAt, &meta, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrNotConnected
		}
		return "", nil, err
	}
	pt.Provider = Provider(prov)
	pt.TokenType = TokenType(tt)
	if extUser != nil {
		pt.ExternalUserID = *extUser
	}
	_ = json.Unmarshal(meta, &pt.Metadata)
	if pt.ExpiresAt != nil && time.Now().After(*pt.ExpiresAt) {
		return "", &pt, ErrTokenExpired
	}
	plain, err := s.cipher.Decrypt(enc)
	if err != nil {
		return "", nil, err
	}
	return string(plain), &pt, nil
}

func (s *Store) GetPlaintextByID(ctx context.Context, tenantID string, tokenID uuid.UUID) (string, *ProviderToken, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, tenant_id, user_subject, provider, token_type, encrypted_token, scopes, external_user_id, expires_at, metadata, created_at, updated_at
FROM provider_tokens WHERE id = $1 AND tenant_id = $2
`, tokenID, tenantID)
	var pt ProviderToken
	var enc []byte
	var prov, tt string
	var meta []byte
	var extUser *string
	err := row.Scan(&pt.ID, &pt.TenantID, &pt.UserSubject, &prov, &tt, &enc, &pt.Scopes, &extUser, &pt.ExpiresAt, &meta, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrNotConnected
		}
		return "", nil, err
	}
	pt.Provider = Provider(prov)
	pt.TokenType = TokenType(tt)
	if extUser != nil {
		pt.ExternalUserID = *extUser
	}
	_ = json.Unmarshal(meta, &pt.Metadata)
	if pt.ExpiresAt != nil && time.Now().After(*pt.ExpiresAt) {
		return "", &pt, ErrTokenExpired
	}
	plain, err := s.cipher.Decrypt(enc)
	if err != nil {
		return "", nil, err
	}
	return string(plain), &pt, nil
}

func (s *Store) ListConnections(ctx context.Context, tenantID, userSubject string) ([]ProviderToken, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, tenant_id, user_subject, provider, token_type, scopes, external_user_id, expires_at, metadata, created_at, updated_at
FROM provider_tokens WHERE tenant_id = $1 AND user_subject = $2
`, tenantID, userSubject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderToken
	for rows.Next() {
		var pt ProviderToken
		var prov, tt string
		var meta []byte
		var extUser *string
		if err := rows.Scan(&pt.ID, &pt.TenantID, &pt.UserSubject, &prov, &tt, &pt.Scopes, &extUser, &pt.ExpiresAt, &meta, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
			return nil, err
		}
		pt.Provider = Provider(prov)
		pt.TokenType = TokenType(tt)
		if extUser != nil {
			pt.ExternalUserID = *extUser
		}
		_ = json.Unmarshal(meta, &pt.Metadata)
		out = append(out, pt)
	}
	return out, rows.Err()
}

func (s *Store) DeleteToken(ctx context.Context, tenantID, userSubject string, provider Provider) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM provider_tokens WHERE tenant_id = $1 AND user_subject = $2 AND provider = $3
`, tenantID, userSubject, string(provider))
	return err
}

func (s *Store) SaveOAuthState(ctx context.Context, state, tenantID, userSubject string, provider Provider, redirectURI string, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO oauth_states (state, tenant_id, user_subject, provider, redirect_uri, expires_at)
VALUES ($1,$2,$3,$4,$5, $6)
`, state, tenantID, userSubject, string(provider), redirectURI, time.Now().Add(ttl))
	return err
}

func (s *Store) ConsumeOAuthState(ctx context.Context, state string) (tenantID, userSubject string, provider Provider, redirectURI string, err error) {
	row := s.pool.QueryRow(ctx, `
DELETE FROM oauth_states WHERE state = $1 AND expires_at > now()
RETURNING tenant_id, user_subject, provider, redirect_uri
`, state)
	var prov string
	err = row.Scan(&tenantID, &userSubject, &prov, &redirectURI)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", "", ErrInvalidOAuthState
		}
		return "", "", "", "", err
	}
	return tenantID, userSubject, Provider(prov), redirectURI, nil
}

func (s *Store) PurgeExpiredOAuthStates(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM oauth_states WHERE expires_at <= now()`)
	return err
}

func (s *Store) UpsertRepoSource(ctx context.Context, repositoryID int64, provider string, externalID, fullName string, tokenID *uuid.UUID, meta map[string]any) error {
	b, _ := json.Marshal(meta)
	_, err := s.pool.Exec(ctx, `
INSERT INTO repo_sources (repository_id, provider, external_repo_id, external_repo_full_name, provider_token_id, access_metadata)
VALUES ($1,$2,$3,$4,$5,$6::jsonb)
ON CONFLICT (repository_id) DO UPDATE SET
  provider = EXCLUDED.provider,
  external_repo_id = EXCLUDED.external_repo_id,
  external_repo_full_name = EXCLUDED.external_repo_full_name,
  provider_token_id = EXCLUDED.provider_token_id,
  access_metadata = EXCLUDED.access_metadata
`, repositoryID, provider, nullStr(externalID), nullStr(fullName), tokenID, b)
	return err
}

var (
	ErrNotConnected      = errors.New("provider not connected")
	ErrTokenExpired      = errors.New("provider token expired")
	ErrInvalidOAuthState = errors.New("invalid or expired oauth state")
)

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// AuthenticatedCloneURL injects credentials for HTTPS git clone (never logged).
func AuthenticatedCloneURL(cloneURL, token string, provider Provider) (string, error) {
	switch provider {
	case ProviderGitHub, ProviderGitLab:
		// https://oauth2:TOKEN@host/owner/repo.git
		if len(cloneURL) < 8 || cloneURL[:8] != "https://" {
			return "", fmt.Errorf("unsupported clone url scheme")
		}
		return "https://oauth2:" + token + "@" + cloneURL[8:], nil
	case ProviderBitbucket:
		// Bitbucket app passwords: x-token-auth:TOKEN
		if len(cloneURL) < 8 || cloneURL[:8] != "https://" {
			return "", fmt.Errorf("unsupported clone url scheme")
		}
		return "https://x-token-auth:" + token + "@" + cloneURL[8:], nil
	default:
		return cloneURL, nil
	}
}

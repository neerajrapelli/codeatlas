package vcsauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"codeatlas/apps/api/internal/config"
	"github.com/google/uuid"
)

type Service struct {
	store *Store
	cfg   config.Config
}

func NewService(store *Store, cfg config.Config) *Service {
	return &Service{store: store, cfg: cfg}
}

func (s *Service) Store() *Store { return s.store }

func (s *Service) OAuthConfig(provider Provider) (OAuthConfig, bool) {
	base := s.cfg.PublicAPIBaseURL
	switch provider {
	case ProviderGitHub:
		if s.cfg.GitHubOAuthClientID == "" {
			return OAuthConfig{}, false
		}
		return OAuthConfig{
			ClientID:     s.cfg.GitHubOAuthClientID,
			ClientSecret: s.cfg.GitHubOAuthClientSecret,
			RedirectURI:  base + "/auth/github/callback",
		}, true
	case ProviderGitLab:
		if s.cfg.GitLabOAuthClientID == "" {
			return OAuthConfig{}, false
		}
		return OAuthConfig{
			ClientID:     s.cfg.GitLabOAuthClientID,
			ClientSecret: s.cfg.GitLabOAuthClientSecret,
			RedirectURI:  base + "/auth/gitlab/callback",
		}, true
	case ProviderBitbucket:
		if s.cfg.BitbucketOAuthClientID == "" {
			return OAuthConfig{}, false
		}
		return OAuthConfig{
			ClientID:     s.cfg.BitbucketOAuthClientID,
			ClientSecret: s.cfg.BitbucketOAuthClientSecret,
			RedirectURI:  base + "/auth/bitbucket/callback",
		}, true
	default:
		return OAuthConfig{}, false
	}
}

func (s *Service) BeginOAuth(ctx context.Context, tenantID, userSubject string, provider Provider) (authorizeURL string, err error) {
	oauthCfg, ok := s.OAuthConfig(provider)
	if !ok {
		return "", fmt.Errorf("oauth not configured for %s", provider)
	}
	state, err := randomState()
	if err != nil {
		return "", err
	}
	if err := s.store.SaveOAuthState(ctx, state, tenantID, userSubject, provider, oauthCfg.RedirectURI, 15*time.Minute); err != nil {
		return "", err
	}
	return AuthorizeURL(provider, oauthCfg, state)
}

func (s *Service) CompleteOAuth(ctx context.Context, provider Provider, state, code string) error {
	tenantID, userSubject, stProvider, redirectURI, err := s.store.ConsumeOAuthState(ctx, state)
	if err != nil {
		return err
	}
	if stProvider != provider {
		return ErrInvalidOAuthState
	}
	oauthCfg, ok := s.OAuthConfig(provider)
	if !ok {
		return fmt.Errorf("oauth not configured")
	}
	oauthCfg.RedirectURI = redirectURI
	access, refresh, expires, scopes, err := ExchangeOAuthCode(ctx, provider, oauthCfg, code)
	if err != nil {
		return err
	}
	meta := map[string]any{}
	if refresh != "" {
		meta["refresh_token"] = refresh
	}
	_, err = s.store.UpsertToken(ctx, ProviderToken{
		TenantID:    tenantID,
		UserSubject: userSubject,
		Provider:    provider,
		TokenType:   TokenOAuth,
		Scopes:      scopes,
		ExpiresAt:   expires,
		Metadata:    meta,
	}, access)
	return err
}

func (s *Service) SavePAT(ctx context.Context, tenantID, userSubject string, provider Provider, token string, tokenType TokenType) error {
	if !provider.Valid() {
		return fmt.Errorf("invalid provider")
	}
	if tokenType != TokenPAT && tokenType != TokenAppPassword {
		tokenType = TokenPAT
	}
	if provider == ProviderBitbucket && tokenType == TokenPAT {
		tokenType = TokenAppPassword
	}
	_, err := s.store.UpsertToken(ctx, ProviderToken{
		TenantID:    tenantID,
		UserSubject: userSubject,
		Provider:    provider,
		TokenType:   tokenType,
		Scopes:      []string{},
	}, token)
	return err
}

func (s *Service) ListRemote(ctx context.Context, tenantID, userSubject string, provider Provider, page int) ([]RemoteRepository, error) {
	plain, _, err := s.store.GetPlaintextToken(ctx, tenantID, userSubject, provider)
	if err != nil {
		return nil, err
	}
	return ListRemoteRepositories(ctx, provider, plain, page)
}

func (s *Service) ResolveCloneURL(ctx context.Context, tenantID string, tokenID *uuid.UUID, provider Provider, cloneURL string) (string, error) {
	if tokenID == nil {
		return cloneURL, nil
	}
	plain, pt, err := s.store.GetPlaintextByID(ctx, tenantID, *tokenID)
	if err != nil {
		return "", err
	}
	if pt.Provider != provider {
		return "", fmt.Errorf("token provider mismatch")
	}
	return AuthenticatedCloneURL(cloneURL, plain, provider)
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

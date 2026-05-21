package repoingest

import (
	"context"
	"errors"

	"codeatlas/apps/api/internal/vcsauth"
	"github.com/google/uuid"
)

// CloneResolver injects provider credentials into HTTPS clone URLs.
type CloneResolver interface {
	ResolveCloneURL(ctx context.Context, tenantID, userSubject string, tokenID *uuid.UUID, sourceType SourceType, cloneURL string) (string, error)
}

type VCSCloneResolver struct {
	vcs *vcsauth.Service
}

func NewVCSCloneResolver(vcs *vcsauth.Service) *VCSCloneResolver {
	return &VCSCloneResolver{vcs: vcs}
}

func (r *VCSCloneResolver) ResolveCloneURL(ctx context.Context, tenantID, userSubject string, tokenID *uuid.UUID, sourceType SourceType, cloneURL string) (string, error) {
	if r == nil || r.vcs == nil {
		return cloneURL, nil
	}
	provider, ok := sourceTypeToProvider(sourceType)
	if !ok {
		return cloneURL, nil
	}
	if tokenID != nil {
		return r.vcs.ResolveCloneURL(ctx, tenantID, tokenID, provider, cloneURL)
	}
	if userSubject == "" {
		return cloneURL, nil
	}
	plain, _, err := r.vcs.Store().GetPlaintextToken(ctx, tenantID, userSubject, provider)
	if err != nil {
		if errors.Is(err, vcsauth.ErrNotConnected) {
			return cloneURL, nil
		}
		return "", err
	}
	return vcsauth.AuthenticatedCloneURL(cloneURL, plain, provider)
}

func sourceTypeToProvider(st SourceType) (vcsauth.Provider, bool) {
	switch st {
	case SourceGitHub:
		return vcsauth.ProviderGitHub, true
	case SourceGitLab:
		return vcsauth.ProviderGitLab, true
	case SourceBitbucket:
		return vcsauth.ProviderBitbucket, true
	default:
		return "", false
	}
}

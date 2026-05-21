// Package repositorysource defines the unified multi-provider repository abstraction.
// Runtime ingestion uses repoingest.Source (Prepare → indexer); this package documents
// the contract and shared types for provider adapters.
package repositorysource

import (
	"context"

	"codeatlas/apps/api/internal/vcsauth"
)

// Provider is the RepositorySource interface: authenticate, list, prepare workspace, stream for indexing.
type Provider interface {
	Type() string
	Authenticate(ctx context.Context, tenantID, userSubject string) error
	ListRepositories(ctx context.Context, tenantID, userSubject string, page int) ([]vcsauth.RemoteRepository, error)
	PrepareWorkspace(ctx context.Context, cloneURL, branch, workspace string) error
}

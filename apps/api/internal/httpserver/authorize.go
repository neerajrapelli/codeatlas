package httpserver

import (
	"context"
	"errors"
	"net/http"

	"codeatlas/apps/api/internal/auth"
	"codeatlas/apps/api/internal/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthorizeRepository ensures the caller's JWT tenant may access the repository.
// When auth is disabled (no claims in context), access is allowed.
func AuthorizeRepository(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) error {
	claims, ok := auth.FromContext(ctx)
	if !ok || claims == nil {
		return nil
	}
	if err := tenant.RepositoryBelongsToTenant(ctx, pool, repositoryID, claims.TenantID); err != nil {
		if errors.Is(err, tenant.ErrForbidden) {
			return tenant.ErrForbidden
		}
		return err
	}
	return nil
}

func writeForbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "repository access denied"})
}

func tenantFromRequest(ctx context.Context) string {
	return tenant.FromContext(ctx)
}

// guardRepository writes 403/404 and returns false when access is denied.
func guardRepository(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, repositoryID int64) bool {
	if err := AuthorizeRepository(r.Context(), pool, repositoryID); err != nil {
		if errors.Is(err, tenant.ErrForbidden) {
			writeForbidden(w)
			return false
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authorization check failed"})
		return false
	}
	return true
}

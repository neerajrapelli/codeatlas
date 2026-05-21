package httpserver

import (
	"context"
	"errors"
	"net/http"

	"codeatlas/apps/api/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errRepositoryForbidden = errors.New("repository access denied")

// AuthorizeRepository ensures the caller's JWT tenant may access the repository.
// When auth is disabled (no claims in context), access is allowed.
func AuthorizeRepository(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) error {
	claims, ok := auth.FromContext(ctx)
	if !ok || claims == nil {
		return nil
	}
	tenant := normalizeTenantID(claims.TenantID)
	var repoTenant string
	err := pool.QueryRow(ctx, `SELECT COALESCE(tenant_id, '') FROM repositories WHERE id = $1`, repositoryID).Scan(&repoTenant)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepositoryForbidden
		}
		return err
	}
	if normalizeTenantID(repoTenant) != tenant {
		return errRepositoryForbidden
	}
	return nil
}

func writeForbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "repository access denied"})
}

func tenantFromRequest(ctx context.Context) string {
	claims, ok := auth.FromContext(ctx)
	if !ok || claims == nil {
		return ""
	}
	return normalizeTenantID(claims.TenantID)
}

func normalizeTenantID(raw string) string {
	if raw == "" {
		return "default"
	}
	return raw
}

// guardRepository writes 403/404 and returns false when access is denied.
func guardRepository(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, repositoryID int64) bool {
	if err := AuthorizeRepository(r.Context(), pool, repositoryID); err != nil {
		if errors.Is(err, errRepositoryForbidden) {
			writeForbidden(w)
			return false
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authorization check failed"})
		return false
	}
	return true
}

package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrForbidden is returned when a repository does not belong to the tenant.
var ErrForbidden = errors.New("repository access denied")

// RepositoryBelongsToTenant checks repositories.tenant_id for the given repo id.
func RepositoryBelongsToTenant(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, tenantID string) error {
	t := Normalize(tenantID)
	var repoTenant string
	err := pool.QueryRow(ctx, `SELECT COALESCE(tenant_id, '') FROM repositories WHERE id = $1`, repositoryID).Scan(&repoTenant)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		return fmt.Errorf("tenant check: %w", err)
	}
	if Normalize(repoTenant) != t {
		return ErrForbidden
	}
	return nil
}

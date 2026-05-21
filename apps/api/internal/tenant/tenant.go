// Package tenant resolves multi-tenant scope from JWT claims.
package tenant

import (
	"context"

	"codeatlas/apps/api/internal/auth"
)

// Normalize returns a non-empty tenant id (default for legacy rows and dev).
func Normalize(raw string) string {
	if raw == "" {
		return "default"
	}
	return raw
}

// FromContext reads tenant_id from JWT claims. Empty string when auth is disabled.
func FromContext(ctx context.Context) string {
	claims, ok := auth.FromContext(ctx)
	if !ok || claims == nil {
		return ""
	}
	return Normalize(claims.TenantID)
}

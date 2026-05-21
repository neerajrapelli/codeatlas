// Package middleware provides HTTP middleware for the CodeAtlas API.
package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"codeatlas/apps/api/internal/auth"
)

// AuthConfig controls JWT enforcement for protected routes.
type AuthConfig struct {
	// Enforced when true (AUTH_DISABLED=false). Missing validator rejects protected routes.
	Enforced  bool
	Validator *auth.Validator
}

// WithAuth validates Bearer tokens (or access_token query for SSE) on protected routes.
func WithAuth(cfg AuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !RequiresAuth(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !cfg.Enforced {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.Validator == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "authentication is not configured (set JWT_SECRET or AUTH_DISABLED=true)",
			})
			return
		}
		token := bearerToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}
		claims, err := cfg.Validator.Validate(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithContext(r.Context(), claims)))
	})
}

// IsPublicRoute returns routes that never require JWT.
func IsPublicRoute(method, path string) bool {
	if method == http.MethodOptions {
		return true
	}
	switch {
	case path == "/health" && method == http.MethodGet:
		return true
	case path == "/metrics" && method == http.MethodGet:
		return true
	case path == "/auth/token" && method == http.MethodPost:
		return true
	case strings.HasSuffix(path, "/callback") && strings.HasPrefix(path, "/auth/") && method == http.MethodGet:
		return true
	default:
		return false
	}
}

// RequiresAuth implements production route policy:
// - all mutating methods (except public routes)
// - all /repositories/* (read + write)
// - /ai/chat
// - /mcp/*
// - /graph/* (architecture data; tenant-scoped handlers)
func RequiresAuth(method, path string) bool {
	if IsPublicRoute(method, path) {
		return false
	}
	if isMutatingMethod(method) {
		return true
	}
	if strings.HasPrefix(path, "/repositories") || strings.HasPrefix(path, "/repos") {
		return true
	}
	if strings.HasPrefix(path, "/ingestion/") {
		return true
	}
	if strings.HasPrefix(path, "/auth/") && !strings.HasSuffix(path, "/callback") {
		return true
	}
	if strings.HasPrefix(path, "/graph") {
		return true
	}
	if path == "/ai/chat" || strings.HasPrefix(path, "/ai/") {
		return true
	}
	if strings.HasPrefix(path, "/mcp") {
		return true
	}
	return false
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func bearerToken(r *http.Request) string {
	raw := r.Header.Get("Authorization")
	if strings.HasPrefix(raw, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	}
	if q := strings.TrimSpace(r.URL.Query().Get("access_token")); q != "" {
		return q
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

package httpserver

import (
	"net/http"
	"strings"

	"codeatlas/apps/api/internal/auth"
)

func isPublicRoute(method, path string) bool {
	if method == http.MethodOptions {
		return true
	}
	if path == "/health" && method == http.MethodGet {
		return true
	}
	if path == "/metrics" && method == http.MethodGet {
		return true
	}
	if path == "/auth/token" && method == http.MethodPost {
		return true
	}
	return false
}

func requiresAuth(method, path string) bool {
	if isPublicRoute(method, path) {
		return false
	}
	return strings.HasPrefix(path, "/repositories") ||
		strings.HasPrefix(path, "/graph") ||
		strings.HasPrefix(path, "/ai") ||
		strings.HasPrefix(path, "/mcp")
}

func withAuth(validator *auth.Validator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if validator == nil || !requiresAuth(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		token := bearerToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}
		claims, err := validator.Validate(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithContext(r.Context(), claims)))
	})
}

// bearerToken reads Authorization header or access_token query (for EventSource/SSE).
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

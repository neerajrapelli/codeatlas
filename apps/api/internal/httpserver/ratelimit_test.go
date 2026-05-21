package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"codeatlas/apps/api/internal/auth"
)

func TestRateLimitMatches(t *testing.T) {
	if !rateLimitMatches("/repositories", "/repositories", http.MethodPost) {
		t.Fatal("expected POST /repositories")
	}
	if !rateLimitMatches("/repositories", "/repositories/1/reindex", http.MethodPost) {
		t.Fatal("expected POST reindex")
	}
	if rateLimitMatches("/repositories", "/repositories/1", http.MethodGet) {
		t.Fatal("GET should not match")
	}
	if !rateLimitMatches("/ai/chat", "/ai/chat", http.MethodPost) {
		t.Fatal("expected POST /ai/chat")
	}
}

func TestRateLimitKey_tenantScoped(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/repositories", nil)
	ctx := auth.WithContext(req.Context(), &auth.Claims{TenantID: "acme"})
	req = req.WithContext(ctx)
	if got := rateLimitKey(req, "/repositories"); got != "tenant:acme:/repositories" {
		t.Fatalf("got %q", got)
	}
}

func TestRateLimitKey_fallsBackToIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ai/chat", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	if got := rateLimitKey(req, "/ai/chat"); got != "ip:203.0.113.5:/ai/chat" {
		t.Fatalf("got %q", got)
	}
}

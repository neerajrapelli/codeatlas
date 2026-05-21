package httpserver

import (
	"net/http"
	"testing"
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

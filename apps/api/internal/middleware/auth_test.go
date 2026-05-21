package middleware

import (
	"net/http"
	"testing"
)

func TestRequiresAuth(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/health", false},
		{http.MethodGet, "/metrics", false},
		{http.MethodPost, "/auth/token", false},
		{http.MethodGet, "/repositories", true},
		{http.MethodPost, "/repositories", true},
		{http.MethodDelete, "/repositories/1", true},
		{http.MethodPost, "/ai/chat", true},
		{http.MethodGet, "/graph/clusters", true},
		{http.MethodPost, "/mcp/tools/foo", true},
		{http.MethodGet, "/mcp/manifest", true},
	}
	for _, tc := range tests {
		if got := RequiresAuth(tc.method, tc.path); got != tc.want {
			t.Errorf("RequiresAuth(%s %s) = %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestWithAuthFailClosed(t *testing.T) {
	cfg := AuthConfig{Enforced: true, Validator: nil}
	h := WithAuth(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req, _ := http.NewRequest(http.MethodPost, "/repositories", nil)
	rr := &recordingResponseWriter{header: make(http.Header)}
	h.ServeHTTP(rr, req)
	if rr.status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.status)
	}
}

type recordingResponseWriter struct {
	header http.Header
	status int
	body   []byte
}

func (r *recordingResponseWriter) Header() http.Header        { return r.header }
func (r *recordingResponseWriter) Write(b []byte) (int, error)  { r.body = append(r.body, b...); return len(b), nil }
func (r *recordingResponseWriter) WriteHeader(statusCode int) { r.status = statusCode }

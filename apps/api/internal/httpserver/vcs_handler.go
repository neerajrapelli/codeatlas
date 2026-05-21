package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"codeatlas/apps/api/internal/auth"
	"codeatlas/apps/api/internal/repoingest"
	"codeatlas/apps/api/internal/tenant"
	"codeatlas/apps/api/internal/vcsauth"
)

func (a *API) registerVCSRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/providers", a.handleAuthProviders)
	for _, p := range []vcsauth.Provider{vcsauth.ProviderGitHub, vcsauth.ProviderGitLab, vcsauth.ProviderBitbucket} {
		provider := p
		mux.HandleFunc("POST /auth/"+string(provider)+"/login", a.handleOAuthLogin(provider))
		mux.HandleFunc("GET /auth/"+string(provider)+"/connect", a.handleOAuthConnect(provider))
		mux.HandleFunc("GET /auth/"+string(provider)+"/callback", a.handleOAuthCallback(provider))
		mux.HandleFunc("POST /auth/"+string(provider)+"/token", a.handleProviderToken(provider))
		mux.HandleFunc("DELETE /auth/"+string(provider)+"/disconnect", a.handleDisconnect(provider))
		mux.HandleFunc("GET /auth/"+string(provider)+"/repositories", a.handleListRemoteRepos(provider))
	}
	mux.HandleFunc("GET /repos", a.handleListRepositories)
	mux.HandleFunc("POST /repos/sync", a.handleReposSync)
	mux.HandleFunc("POST /repos/upload-zip", a.handleUploadZip)
	mux.HandleFunc("GET /ingestion/jobs/{jobId}", a.handleIngestionJobByID)
}

func (a *API) handleAuthProviders(w http.ResponseWriter, r *http.Request) {
	if a.vcs == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vcs auth unavailable"})
		return
	}
	tenantID, subject, ok := authSubject(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	conns, err := a.vcs.Store().ListConnections(ctx, tenantID, subject)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list connections"})
		return
	}
	type row struct {
		Provider       string     `json:"provider"`
		TokenType      string     `json:"tokenType"`
		Scopes         []string   `json:"scopes"`
		ExternalUserID string     `json:"externalUserId,omitempty"`
		ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
		Connected      bool       `json:"connected"`
	}
	out := make([]row, 0, len(conns))
	for _, c := range conns {
		out = append(out, row{
			Provider:       string(c.Provider),
			TokenType:      string(c.TokenType),
			Scopes:         c.Scopes,
			ExternalUserID: c.ExternalUserID,
			ExpiresAt:      c.ExpiresAt,
			Connected:      true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": out})
}

func (a *API) handleOAuthLogin(provider vcsauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.beginOAuthJSON(w, r, provider)
	}
}

func (a *API) handleOAuthConnect(provider vcsauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("redirect") == "1" {
			url, err := a.oauthAuthorizeURL(r, provider)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			http.Redirect(w, r, url, http.StatusFound)
			return
		}
		a.beginOAuthJSON(w, r, provider)
	}
}

func (a *API) beginOAuthJSON(w http.ResponseWriter, r *http.Request, provider vcsauth.Provider) {
	if a.vcs == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vcs auth unavailable"})
		return
	}
	tenantID, subject, ok := authSubject(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	url, err := a.vcs.BeginOAuth(ctx, tenantID, subject, provider)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorizeUrl": url})
}

func (a *API) oauthAuthorizeURL(r *http.Request, provider vcsauth.Provider) (string, error) {
	tenantID, subject, ok := authSubject(r)
	if !ok {
		return "", errors.New("authentication required")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	return a.vcs.BeginOAuth(ctx, tenantID, subject, provider)
}

func (a *API) handleOAuthCallback(provider vcsauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.vcs == nil {
			http.Error(w, "vcs auth unavailable", http.StatusServiceUnavailable)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			http.Redirect(w, r, a.cfg.FrontendURL+"/?vcs_error="+errMsg, http.StatusFound)
			return
		}
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if state == "" || code == "" {
			http.Error(w, "missing state or code", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if err := a.vcs.CompleteOAuth(ctx, provider, state, code); err != nil {
			slog.Error("oauth_callback_failed", "provider", provider, "error", err)
			http.Redirect(w, r, a.cfg.FrontendURL+"/?vcs_error=oauth_failed", http.StatusFound)
			return
		}
		http.Redirect(w, r, a.cfg.FrontendURL+"/?vcs_connected="+string(provider), http.StatusFound)
	}
}

func (a *API) handleProviderToken(provider vcsauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.vcs == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vcs auth unavailable"})
			return
		}
		tenantID, subject, ok := authSubject(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		var body struct {
			Token     string `json:"token"`
			TokenType string `json:"tokenType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token is required"})
			return
		}
		tt := vcsauth.TokenPAT
		if body.TokenType == "app_password" {
			tt = vcsauth.TokenAppPassword
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := a.vcs.SavePAT(ctx, tenantID, subject, provider, body.Token, tt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "connected", "provider": string(provider)})
	}
}

func (a *API) handleDisconnect(provider vcsauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.vcs == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vcs auth unavailable"})
			return
		}
		tenantID, subject, ok := authSubject(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := a.vcs.Store().DeleteToken(ctx, tenantID, subject, provider); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "disconnect failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
	}
}

func (a *API) handleListRemoteRepos(provider vcsauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.vcs == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vcs auth unavailable"})
			return
		}
		tenantID, subject, ok := authSubject(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		repos, err := a.vcs.ListRemote(ctx, tenantID, subject, provider, page)
		if err != nil {
			if errors.Is(err, vcsauth.ErrNotConnected) || errors.Is(err, vcsauth.ErrTokenExpired) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repositories": repos})
	}
}

func (a *API) handleUploadZip(w http.ResponseWriter, r *http.Request) {
	a.handleCreateRepository(w, r)
}

func (a *API) handleReposSync(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
		return
	}
	var body struct {
		RepositoryID int64 `json:"repositoryId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RepositoryID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repositoryId is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := a.ingest.Reindex(ctx, body.RepositoryID); err != nil {
		if errors.Is(err, repoingest.ErrRepositoryNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reindex_started"})
}

func (a *API) handleIngestionJobByID(w http.ResponseWriter, r *http.Request) {
	if a.ingestQueue == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "job queue unavailable"})
		return
	}
	jobID := r.PathValue("jobId")
	if jobID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "jobId required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	job, err := a.ingestQueue.GetByID(ctx, jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load job"})
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"jobId":        job.ID,
		"repositoryId": job.RepositoryID,
		"status":       job.Status,
		"phase":        job.Phase,
		"currentStep":  job.CurrentStep,
		"progress":     job.Progress,
		"metadata":     job.Metadata,
		"error":        job.ErrorMsg,
		"queuedAt":     job.QueuedAt,
		"startedAt":    job.StartedAt,
		"completedAt":  job.CompletedAt,
	})
}

func authSubject(r *http.Request) (tenantID, subject string, ok bool) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims == nil {
		return "", "", false
	}
	subject = claims.Sub
	if subject == "" {
		subject = claims.Subject
	}
	tenantID = tenant.Normalize(claims.TenantID)
	if tenantID == "" {
		tenantID = "default"
	}
	return tenantID, subject, true
}

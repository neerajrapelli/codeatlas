package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"codeatlas/apps/api/internal/auth"
	"codeatlas/apps/api/internal/ingestprogress"
	"codeatlas/apps/api/internal/repoingest"
	"codeatlas/apps/api/internal/socio"
	"codeatlas/apps/api/internal/tenant"
)

func (a *API) registerIngestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /repositories", a.handleListRepositories)
	mux.HandleFunc("GET /repositories/{id}/ingestion/status", a.handleIngestionStatus)
	mux.HandleFunc("GET /repositories/{id}/ingestion/stream", a.handleIngestionStream)
	mux.HandleFunc("POST /repositories", a.handleCreateRepository)
	mux.HandleFunc("DELETE /repositories/{id}", a.handleDeleteRepository)
	mux.HandleFunc("POST /repositories/{id}/reindex", a.handleReindexRepository)
}

func (a *API) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	repos, err := a.ingest.ListRecent(ctx, tenantFromRequest(r.Context()), 25)
	if err != nil {
		slog.Error("repositories_list_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list repositories"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositories": repos})
}

func (a *API) handleIngestionStatus(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil || a.socioQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion status unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var code socio.CodeIndexStatus
	if a.ingestQueue != nil {
		job, err := a.ingestQueue.GetStatus(ctx, strconv.FormatInt(repoID, 10))
		if err != nil {
			slog.Error("ingestion_job_status_failed", "repository_id", repoID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load ingestion job"})
			return
		}
		if job != nil {
			code = streamEventToCodeIndex(job.Progress)
		}
	}
	if code.Status == "" {
		progress, err := a.ingest.GetProgress(ctx, repoID)
		if err != nil {
			slog.Error("ingestion_status_progress_failed", "repository_id", repoID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load ingestion status"})
			return
		}
		code = socio.CodeIndexStatus{
			Status:          string(progress.Status),
			Stage:           string(progress.Stage),
			ProgressPercent: progress.ProgressPercent,
			FilesIndexed:    progress.Metrics.FilesIndexed,
		}
	}
	status, err := a.socioQuery.BuildIngestionStatus(ctx, repoID, code)
	if err != nil {
		slog.Error("ingestion_status_failed", "repository_id", repoID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build ingestion status"})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (a *API) handleIngestionStream(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion status unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	progressTicker := time.NewTicker(500 * time.Millisecond)
	defer progressTicker.Stop()
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	lastJSON := ""
	sendEvent := func() bool {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		ev, err := a.ingest.GetIngestionStreamEvent(ctx, repoID)
		if err != nil {
			return false
		}
		raw, err := json.Marshal(ev)
		if err != nil {
			return false
		}
		if string(raw) == lastJSON {
			return false
		}
		lastJSON = string(raw)
		if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
			return true
		}
		flusher.Flush()
		return ev.Status == ingestprogress.StatusComplete || ev.Status == ingestprogress.StatusFailed
	}

	if sendEvent() {
		return
	}

	var redisCh <-chan ingestprogress.StreamEvent
	var redisCleanup func()
	if a.progressBus != nil {
		if ch, cleanup, err := a.progressBus.Subscribe(r.Context(), repoID); err == nil && ch != nil {
			redisCh = ch
			redisCleanup = cleanup
			defer redisCleanup()
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-redisCh:
			if !ok {
				redisCh = nil
				continue
			}
			raw, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if string(raw) != lastJSON {
				lastJSON = string(raw)
				if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
					return
				}
				flusher.Flush()
				if ev.Status == ingestprogress.StatusComplete || ev.Status == ingestprogress.StatusFailed {
					return
				}
			}
		case <-heartbeatTicker.C:
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-progressTicker.C:
			if sendEvent() {
				return
			}
		}
	}
}

func (a *API) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
		return
	}
	req, cleanup, err := parseRepositoryCreateRequest(r, a.cfg.ZipMaxBytes)
	_ = cleanup
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	req.TenantID = tenant.Normalize(tenantFromRequest(r.Context()))
	if claims, ok := auth.FromContext(r.Context()); ok && claims != nil {
		req.UserSubject = claims.Sub
		if req.UserSubject == "" {
			req.UserSubject = claims.Subject
		}
	}
	repo, jobID, err := a.ingest.Enqueue(ctx, req, req.TenantID)
	if err != nil {
		slog.Error("repository_ingestion_failed", "error", err, "source_type", req.SourceType, "source_url", req.SourceURL)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"repository": repo, "jobId": jobID})
}

func (a *API) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	// Large repos can take minutes to cascade-delete indexed rows.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	repo, err := a.ingest.Delete(ctx, repoID)
	if err != nil {
		if errors.Is(err, repoingest.ErrRepositoryNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("repository_delete_failed", "error", err, "repository_id", repoID)
			writeJSON(w, http.StatusGatewayTimeout, map[string]string{
				"error": "repository delete timed out; try again or contact support if it persists",
			})
			return
		}
		slog.Error("repository_delete_failed", "error", err, "repository_id", repoID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete repository"})
		return
	}
	canRestore := repo.SourceType != repoingest.SourceZIP
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
		"undo": map[string]any{
			"sourceType":  repo.SourceType,
			"sourceUrl":   repo.SourceURL,
			"branch":      repo.Branch,
			"displayName": repo.Name,
			"canRestore":  canRestore,
		},
	})
}

func (a *API) handleReindexRepository(w http.ResponseWriter, r *http.Request) {
	if a.ingest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := a.ingest.Reindex(ctx, repoID); err != nil {
		if errors.Is(err, repoingest.ErrRepositoryNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
			return
		}
		if errors.Is(err, repoingest.ErrZIPWorkspaceMissing) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		slog.Error("repository_reindex_failed", "repository_id", repoID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reindex_started"})
}

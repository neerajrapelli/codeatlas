package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *API) registerArchitectureRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /repositories/{id}/architecture/timeline", a.handleArchitectureTimeline)
	mux.HandleFunc("GET /repositories/{id}/architecture/decisions", a.handleArchitectureDecisions)
	mux.HandleFunc("GET /repositories/{id}/architecture/module-intel", a.handleArchitectureModuleIntel)
	mux.HandleFunc("GET /repositories/{id}/architecture/pr-insights", a.handleArchitecturePRInsights)
	mux.HandleFunc("GET /repositories/{id}/architecture/maintainer-influence", a.handleArchitectureMaintainerInfluence)
	mux.HandleFunc("GET /repositories/{id}/architecture/search", a.handleArchitectureSearch)
}

func parseLimit(r *http.Request, def, max int) int {
	limit := def
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > max {
				n = max
			}
			limit = n
		}
	}
	return limit
}

func (a *API) handleArchitectureTimeline(w http.ResponseWriter, r *http.Request) {
	if a.archQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "architecture timeline unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	limit := parseLimit(r, 200, 1000)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := a.archQuery.ListTimeline(ctx, repoID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load timeline"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositoryId": repoID, "items": items})
}

func (a *API) handleArchitectureDecisions(w http.ResponseWriter, r *http.Request) {
	if a.archQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "architecture decisions unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	limit := parseLimit(r, 200, 1000)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := a.archQuery.ListDecisions(ctx, repoID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load decisions"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositoryId": repoID, "items": items})
}

func (a *API) handleArchitectureModuleIntel(w http.ResponseWriter, r *http.Request) {
	if a.archQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "module intelligence unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	modulePath := strings.TrimSpace(r.URL.Query().Get("path"))
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	item, err := a.archQuery.GetModuleIntelligence(ctx, repoID, modulePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load module intelligence"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleArchitecturePRInsights(w http.ResponseWriter, r *http.Request) {
	if a.archQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pr insights unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	limit := parseLimit(r, 50, 500)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := a.archQuery.ListPRInsights(ctx, repoID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load pr insights"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositoryId": repoID, "items": items})
}

func (a *API) handleArchitectureMaintainerInfluence(w http.ResponseWriter, r *http.Request) {
	if a.archQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "maintainer influence unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	limit := parseLimit(r, 20, 200)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := a.archQuery.ListMaintainerInfluence(ctx, repoID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load maintainer influence"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositoryId": repoID, "items": items})
}

func (a *API) handleArchitectureSearch(w http.ResponseWriter, r *http.Request) {
	if a.archQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "architecture search unavailable"})
		return
	}
	repoID, ok := parsePathRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q is required"})
		return
	}
	limit := parseLimit(r, 30, 200)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := a.archQuery.Search(ctx, repoID, query, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to search architecture intelligence"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositoryId": repoID, "query": query, "items": items})
}

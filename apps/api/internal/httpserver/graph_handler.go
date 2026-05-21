package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"codeatlas/apps/api/internal/graphhierarchy"
)

func (a *API) registerGraphRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /graph/files", a.handleGraphFiles)
	mux.HandleFunc("GET /graph/clusters", a.handleGraphClusters)
	mux.HandleFunc("GET /graph/file", a.handleGraphFile)
	mux.HandleFunc("GET /graph/symbols", a.handleGraphSymbols)
}

func (a *API) handleGraphFiles(w http.ResponseWriter, r *http.Request) {
	repoID, ok := parseRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	page := parseGraphPage(r, a.cfg.GraphMaxFileLimit, a.cfg.GraphMaxFileLimit*4)

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	files, deps, total, truncated, err := loadGraphPayload(ctx, a.pool, repoID, tenantFromRequest(r.Context()), page)
	if err != nil {
		slog.Error("graph_query_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load graph"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"files":        files,
		"dependencies": deps,
		"total":        total,
		"limit":        page.Limit,
		"offset":       page.Offset,
		"truncated":    truncated,
	})
}

func (a *API) handleGraphClusters(w http.ResponseWriter, r *http.Request) {
	repoID, ok := parseRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	maxDepth := parseGraphDepth(r, a.cfg.GraphMaxDepth, a.cfg.GraphMaxDepth)

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	layer, err := graphhierarchy.BuildLayer(ctx, a.pool, repoID, tenantFromRequest(r.Context()), prefix, maxDepth)
	if err != nil {
		slog.Error("graph_clusters_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build cluster layer"})
		return
	}
	files := layer.Files
	if len(files) > maxClusterFiles {
		files = files[:maxClusterFiles]
	}
	edges := layer.Edges
	if len(edges) > maxClusterEdges {
		edges = edges[:maxClusterEdges]
	}
	payload := map[string]any{
		"prefix":     layer.Prefix,
		"clusters":   layer.Clusters,
		"files":      files,
		"edges":      edges,
		"maxDepth":   maxDepth,
		"truncated":  len(layer.Files) > maxClusterFiles || len(layer.Edges) > maxClusterEdges,
		"totalFiles": len(layer.Files),
		"totalEdges": len(layer.Edges),
	}
	if a.socioQuery != nil {
		if overlay, err := a.socioQuery.GetFileOverlays(ctx, repoID); err == nil {
			payload["socioOverlay"] = overlay
		}
	}
	writeJSON(w, http.StatusOK, payload)
}

func (a *API) handleGraphFile(w http.ResponseWriter, r *http.Request) {
	repoID, ok := parseRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	raw := r.URL.Query().Get("fileId")
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fileId is required"})
		return
	}
	fileID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || fileID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid fileId"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	gf, err := loadSingleGraphFile(ctx, a.pool, repoID, fileID, tenantFromRequest(r.Context()))
	if err != nil {
		slog.Error("graph_file_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load file"})
		return
	}
	writeJSON(w, http.StatusOK, gf)
}

func (a *API) handleGraphSymbols(w http.ResponseWriter, r *http.Request) {
	repoID, ok := parseRepositoryIDGuarded(w, r, a.pool)
	if !ok {
		return
	}
	raw := r.URL.Query().Get("fileId")
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fileId is required"})
		return
	}
	fileID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || fileID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid fileId"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	symbols, err := loadSymbolsForFile(ctx, a.pool, repoID, fileID, tenantFromRequest(r.Context()))
	if err != nil {
		slog.Error("graph_symbols_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load symbols"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbols": symbols})
}

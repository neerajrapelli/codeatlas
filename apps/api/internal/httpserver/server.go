package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"codeatlas/apps/api/internal/ai"
	"codeatlas/apps/api/internal/config"
	"codeatlas/apps/api/internal/graphhierarchy"
	"codeatlas/apps/api/internal/repoingest"
	"github.com/jackc/pgx/v5/pgxpool"
)

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

func New(cfg config.Config, pool *pgxpool.Pool, aiService *ai.Service, ingestService *repoingest.Service) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{
			Service: "codeatlas-api",
			Status:  "ok",
			Version: "0.1.0",
		})
	})

	mux.HandleFunc("GET /graph/files", func(w http.ResponseWriter, r *http.Request) {
		repoID, err := parseRepositoryID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		files, deps, err := loadGraphPayload(ctx, pool, repoID)
		if err != nil {
			slog.Error("graph_query_failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load graph"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"files": files, "dependencies": deps})
	})

	mux.HandleFunc("GET /graph/clusters", func(w http.ResponseWriter, r *http.Request) {
		repoID, err := parseRepositoryID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		layer, err := graphhierarchy.BuildLayer(ctx, pool, repoID, prefix)
		if err != nil {
			slog.Error("graph_clusters_failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build cluster layer"})
			return
		}
		writeJSON(w, http.StatusOK, layer)
	})

	mux.HandleFunc("GET /graph/file", func(w http.ResponseWriter, r *http.Request) {
		repoID, err := parseRepositoryID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
		gf, err := loadSingleGraphFile(ctx, pool, repoID, fileID)
		if err != nil {
			slog.Error("graph_file_failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load file"})
			return
		}
		writeJSON(w, http.StatusOK, gf)
	})

	mux.HandleFunc("GET /graph/symbols", func(w http.ResponseWriter, r *http.Request) {
		repoID, err := parseRepositoryID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
		symbols, err := loadSymbolsForFile(ctx, pool, repoID, fileID)
		if err != nil {
			slog.Error("graph_symbols_failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load symbols"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"symbols": symbols})
	})

	mux.HandleFunc("GET /repositories", func(w http.ResponseWriter, r *http.Request) {
		if ingestService == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		repos, err := ingestService.ListRecent(ctx, 25)
		if err != nil {
			slog.Error("list_repositories_failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list repositories"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repositories": repos})
	})

	mux.HandleFunc("GET /repositories/{id}/progress", func(w http.ResponseWriter, r *http.Request) {
		if ingestService == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
			return
		}
		repoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || repoID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid repository id"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		progress, err := ingestService.GetProgress(ctx, repoID)
		if err != nil {
			slog.Error("repository_progress_failed", "repository_id", repoID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load repository progress"})
			return
		}
		writeJSON(w, http.StatusOK, progress)
	})

	mux.HandleFunc("POST /repositories", func(w http.ResponseWriter, r *http.Request) {
		if ingestService == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
			return
		}
		req, cleanup, err := parseRepositoryCreateRequest(r, cfg.ZipMaxBytes)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		repo, err := ingestService.Enqueue(ctx, req)
		if err != nil {
			slog.Error("repository_ingestion_failed", "error", err, "source_type", req.SourceType, "source_url", req.SourceURL)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, repo)
	})

	mux.HandleFunc("DELETE /repositories/{id}", func(w http.ResponseWriter, r *http.Request) {
		if ingestService == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
			return
		}
		repoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || repoID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid repository id"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		repo, err := ingestService.Delete(ctx, repoID)
		if err != nil {
			if errors.Is(err, repoingest.ErrRepositoryNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
				return
			}
			slog.Error("repository_delete_failed", "error", err)
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
	})

	mux.HandleFunc("POST /repositories/{id}/reindex", func(w http.ResponseWriter, r *http.Request) {
		if ingestService == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingestion unavailable"})
			return
		}
		repoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || repoID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid repository id"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := ingestService.Reindex(ctx, repoID); err != nil {
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
	})

	mux.HandleFunc("POST /ai/chat", func(w http.ResponseWriter, r *http.Request) {
		if aiService == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI service is not configured"})
			return
		}

		var req ai.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if req.RepositoryID == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repositoryId is required"})
			return
		}

		if !req.Stream {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			resp, err := aiService.Answer(ctx, req)
			if err != nil {
				slog.Error("ai_chat_failed", "error", err)
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()

		prepared, err := aiService.PrepareChat(ctx, req)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		if prepared.ContextFileCount == 0 {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			flusher, ok := w.(http.Flusher)
			if !ok {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
				return
			}
			_ = writeSSE(w, map[string]any{
				"type":         "meta",
				"relatedFiles": prepared.RelatedFiles,
				"provider":     string(prepared.Provider),
				"model":        prepared.Model,
			})
			flusher.Flush()
			_ = writeSSE(w, map[string]any{"type": "token", "token": "I could not find relevant indexed context for this repository yet."})
			_ = writeSSE(w, map[string]any{"type": "done"})
			flusher.Flush()
			return
		}

		chunks, chunkErrs, err := aiService.StreamCompletion(ctx, prepared)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
			return
		}

		if err := writeSSE(w, map[string]any{
			"type":         "meta",
			"relatedFiles": prepared.RelatedFiles,
			"provider":     string(prepared.Provider),
			"model":        prepared.Model,
		}); err != nil {
			return
		}
		flusher.Flush()

		for chunks != nil || chunkErrs != nil {
			select {
			case err, ok := <-chunkErrs:
				if !ok {
					chunkErrs = nil
					continue
				}
				if err != nil {
					_ = writeSSE(w, map[string]any{"type": "error", "error": err.Error()})
					flusher.Flush()
					return
				}
			case ch, ok := <-chunks:
				if !ok {
					chunks = nil
					continue
				}
				if ch.Delta != "" {
					if err := writeSSE(w, map[string]any{"type": "token", "token": ch.Delta}); err != nil {
						return
					}
					flusher.Flush()
				}
				if ch.Done {
					_ = writeSSE(w, map[string]any{"type": "done"})
					flusher.Flush()
					return
				}
			}
		}
		_ = writeSSE(w, map[string]any{"type": "done"})
		flusher.Flush()
	})

	handler := withCORS(cfg.AllowedOrigins, loggingMiddleware(mux))
	return &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
}

func parseRepositoryCreateRequest(r *http.Request, zipMaxBytes int64) (repoingest.CreateRequest, func(), error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parseMultipartRepositoryRequest(r, zipMaxBytes)
	}
	var req repoingest.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, nil, fmt.Errorf("invalid JSON body")
	}
	if req.SourceType == "" {
		return req, nil, fmt.Errorf("sourceType is required")
	}
	if req.SourceType == repoingest.SourceZIP {
		return req, nil, fmt.Errorf("zip uploads must use multipart/form-data")
	}
	if req.SourceURL == "" {
		return req, nil, fmt.Errorf("sourceUrl is required")
	}
	return req, nil, nil
}

func parseMultipartRepositoryRequest(r *http.Request, zipMaxBytes int64) (repoingest.CreateRequest, func(), error) {
	const maxMultipart = int64(128 << 20)
	if err := r.ParseMultipartForm(maxMultipart); err != nil {
		return repoingest.CreateRequest{}, nil, fmt.Errorf("invalid multipart form")
	}
	sourceType := repoingest.SourceType(r.FormValue("sourceType"))
	if sourceType == "" {
		sourceType = repoingest.SourceZIP
	}
	if sourceType != repoingest.SourceZIP {
		return repoingest.CreateRequest{}, nil, fmt.Errorf("multipart currently supports zip sourceType only")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return repoingest.CreateRequest{}, nil, fmt.Errorf("zip file is required")
	}
	defer file.Close()
	if filepath.Ext(header.Filename) != ".zip" {
		return repoingest.CreateRequest{}, nil, fmt.Errorf("unsupported archive; only .zip is allowed")
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("codeatlas-upload-%d.zip", time.Now().UnixNano()))
	out, err := os.Create(tmpPath)
	if err != nil {
		return repoingest.CreateRequest{}, nil, fmt.Errorf("create temp zip: %w", err)
	}
	written, copyErr := io.Copy(out, io.LimitReader(file, zipMaxBytes+1))
	_ = out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return repoingest.CreateRequest{}, nil, fmt.Errorf("save uploaded zip: %w", copyErr)
	}
	if written > zipMaxBytes {
		_ = os.Remove(tmpPath)
		return repoingest.CreateRequest{}, nil, fmt.Errorf("zip exceeds max size (%d bytes)", zipMaxBytes)
	}

	req := repoingest.CreateRequest{
		SourceType:  repoingest.SourceZIP,
		DisplayName: r.FormValue("displayName"),
		ZIPPath:     tmpPath,
	}
	return req, func() { _ = os.Remove(tmpPath) }, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSSE(w http.ResponseWriter, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}

func parseRepositoryID(r *http.Request) (int64, error) {
	raw := r.URL.Query().Get("repositoryId")
	if raw == "" {
		return 1, nil
	}
	repoID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || repoID <= 0 {
		return 0, errors.New("repositoryId must be a positive integer")
	}
	return repoID, nil
}

type graphFile struct {
	ID      string            `json:"id"`
	Path    string            `json:"path"`
	Imports []string          `json:"imports"`
	Exports []string          `json:"exports"`
	Symbols []map[string]any  `json:"symbols"`
}

type graphDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func loadGraphPayload(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) ([]graphFile, []graphDependency, error) {
	fileRows, err := pool.Query(ctx, `
		SELECT
			f.id,
			f.relative_path,
			COALESCE(array_agg(DISTINCT fi.module_path) FILTER (WHERE fi.module_path IS NOT NULL), '{}') AS imports,
			COALESCE(array_agg(DISTINCT fe.export_name) FILTER (WHERE fe.export_name IS NOT NULL), '{}') AS exports
		FROM files f
		LEFT JOIN file_imports fi ON fi.file_id = f.id
		LEFT JOIN file_exports fe ON fe.file_id = f.id
		WHERE f.repository_id = $1
		GROUP BY f.id, f.relative_path
		ORDER BY f.relative_path
	`, repositoryID)
	if err != nil {
		return nil, nil, err
	}
	defer fileRows.Close()

	files := make([]graphFile, 0, 256)
	fileIDToNode := make(map[int64]string, 256)
	for fileRows.Next() {
		var id int64
		var path string
		var imports []string
		var exports []string
		if err := fileRows.Scan(&id, &path, &imports, &exports); err != nil {
			return nil, nil, err
		}
		nodeID := strconv.FormatInt(id, 10)
		fileIDToNode[id] = nodeID
		files = append(files, graphFile{
			ID:      nodeID,
			Path:    path,
			Imports: imports,
			Exports: exports,
			Symbols: make([]map[string]any, 0),
		})
	}
	if err := fileRows.Err(); err != nil {
		return nil, nil, err
	}

	symbolRows, err := pool.Query(ctx, `
		SELECT file_id, name, kind
		FROM symbols
		WHERE repository_id = $1
	`, repositoryID)
	if err != nil {
		return nil, nil, err
	}
	defer symbolRows.Close()

	symbolsByNode := make(map[string][]map[string]any, len(files))
	for symbolRows.Next() {
		var fileID int64
		var name, kind string
		if err := symbolRows.Scan(&fileID, &name, &kind); err != nil {
			return nil, nil, err
		}
		nodeID, ok := fileIDToNode[fileID]
		if !ok {
			continue
		}
		symbolsByNode[nodeID] = append(symbolsByNode[nodeID], map[string]any{"name": name, "kind": kind})
	}
	if err := symbolRows.Err(); err != nil {
		return nil, nil, err
	}
	for i := range files {
		files[i].Symbols = symbolsByNode[files[i].ID]
	}

	depRows, err := pool.Query(ctx, `
		SELECT from_file_id, to_file_id
		FROM file_dependencies
		WHERE repository_id = $1
	`, repositoryID)
	if err != nil {
		return nil, nil, err
	}
	defer depRows.Close()

	deps := make([]graphDependency, 0, 512)
	for depRows.Next() {
		var fromID int64
		var toID int64
		if err := depRows.Scan(&fromID, &toID); err != nil {
			return nil, nil, err
		}
		fromNode, okFrom := fileIDToNode[fromID]
		toNode, okTo := fileIDToNode[toID]
		if !okFrom || !okTo {
			continue
		}
		deps = append(deps, graphDependency{From: fromNode, To: toNode})
	}
	if err := depRows.Err(); err != nil {
		return nil, nil, err
	}
	return files, deps, nil
}

func loadSingleGraphFile(ctx context.Context, pool *pgxpool.Pool, repositoryID, fileID int64) (graphFile, error) {
	var id int64
	var path string
	var imports []string
	var exports []string
	err := pool.QueryRow(ctx, `
		SELECT f.id, f.relative_path,
			COALESCE(array_agg(DISTINCT fi.module_path) FILTER (WHERE fi.module_path IS NOT NULL), '{}'),
			COALESCE(array_agg(DISTINCT fe.export_name) FILTER (WHERE fe.export_name IS NOT NULL), '{}')
		FROM files f
		LEFT JOIN file_imports fi ON fi.file_id = f.id
		LEFT JOIN file_exports fe ON fe.file_id = f.id
		WHERE f.repository_id = $1 AND f.id = $2
		GROUP BY f.id, f.relative_path
	`, repositoryID, fileID).Scan(&id, &path, &imports, &exports)
	if err != nil {
		return graphFile{}, err
	}
	nodeID := strconv.FormatInt(id, 10)
	symbols, err := loadSymbolsForFile(ctx, pool, repositoryID, fileID)
	if err != nil {
		return graphFile{}, err
	}
	symbolMaps := make([]map[string]any, 0, len(symbols))
	for _, s := range symbols {
		symbolMaps = append(symbolMaps, map[string]any{"name": s.Name, "kind": s.Kind})
	}
	return graphFile{
		ID:      nodeID,
		Path:    path,
		Imports: imports,
		Exports: exports,
		Symbols: symbolMaps,
	}, nil
}

type symbolRow struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func loadSymbolsForFile(ctx context.Context, pool *pgxpool.Pool, repositoryID, fileID int64) ([]symbolRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT name, kind FROM symbols WHERE repository_id = $1 AND file_id = $2 ORDER BY start_line
	`, repositoryID, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]symbolRow, 0, 32)
	for rows.Next() {
		var s symbolRow
		if err := rows.Scan(&s.Name, &s.Kind); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	allow := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allow[o] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allow[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

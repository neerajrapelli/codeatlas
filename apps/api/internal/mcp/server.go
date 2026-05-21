package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeatlas/apps/api/internal/blastradius"
	"codeatlas/apps/api/internal/driftdetector"
	"codeatlas/apps/api/internal/socio"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server exposes CodeAtlas intelligence via MCP-style HTTP tools.
type Server struct {
	pool        *pgxpool.Pool
	blast       *blastradius.Service
	drift       *driftdetector.Engine
	socio       *socio.QueryService
	log         *callLog
}

func NewServer(pool *pgxpool.Pool, blast *blastradius.Service, drift *driftdetector.Engine, socio *socio.QueryService) *Server {
	return &Server{
		pool:  pool,
		blast: blast,
		drift: drift,
		socio: socio,
		log:   newCallLog(100),
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /mcp/manifest", s.handleManifest)
	mux.HandleFunc("GET /mcp/logs", s.handleLogs)
	mux.HandleFunc("POST /mcp/tools/{tool_name}", s.handleTool)
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, manifest())
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": s.log.recent(limit)})
}

func (s *Server) handleTool(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool_name")
	var params map[string]any
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, meta, err := s.execute(ctx, tool, params)
	entry := logEntry{Tool: tool, At: time.Now().UTC(), OK: err == nil}
	if err != nil {
		entry.Error = err.Error()
		s.log.add(entry)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	entry.RepositoryID = meta.RepositoryID
	s.log.add(entry)
	writeJSON(w, http.StatusOK, map[string]any{
		"tool":   tool,
		"result": result,
		"meta":   meta,
	})
}

type toolMeta struct {
	RepositoryID      string    `json:"repository_id"`
	GeneratedAt       time.Time `json:"generated_at"`
	GraphCompleteness float64   `json:"graph_completeness"`
}

func (s *Server) execute(ctx context.Context, tool string, params map[string]any) (any, toolMeta, error) {
	repoID, err := parseRepoID(params["repository_id"])
	if err != nil {
		return nil, toolMeta{}, err
	}
	meta := toolMeta{
		RepositoryID:      strconv.FormatInt(repoID, 10),
		GeneratedAt:       time.Now().UTC(),
		GraphCompleteness: s.graphCompleteness(ctx, repoID),
	}
	switch tool {
	case "get_file_context":
		path, _ := params["file_path"].(string)
		if path == "" {
			return nil, meta, fmt.Errorf("file_path is required")
		}
		res, err := s.fileContext(ctx, repoID, path)
		return res, meta, err
	case "get_blast_radius":
		path, _ := params["file_path"].(string)
		if path == "" {
			return nil, meta, fmt.Errorf("file_path is required")
		}
		sym, _ := params["symbol_name"].(string)
		if s.blast == nil {
			return nil, meta, fmt.Errorf("blast radius unavailable")
		}
		res, err := s.blast.Analyze(ctx, repoID, path, sym, 3)
		return res, meta, err
	case "get_ownership":
		path, _ := params["file_path"].(string)
		if path == "" {
			return nil, meta, fmt.Errorf("file_path is required")
		}
		if s.socio == nil {
			return nil, meta, fmt.Errorf("ownership unavailable")
		}
		rows, err := s.socio.GetOwnership(ctx, repoID, 0)
		if err != nil {
			return nil, meta, err
		}
		for _, row := range rows {
			if row.Path == path || strings.HasSuffix(row.Path, "/"+path) {
				return row, meta, nil
			}
		}
		return map[string]any{"path": path, "message": "no ownership data"}, meta, nil
	case "get_hotspots":
		limit := 10
		if n, ok := params["limit"].(float64); ok && n > 0 {
			limit = int(n)
		}
		if s.socio == nil {
			return nil, meta, fmt.Errorf("hotspots unavailable")
		}
		rows, err := s.socio.GetHotspots(ctx, repoID, limit)
		return map[string]any{"hotspots": rows}, meta, err
	case "get_architecture_signals":
		rows, err := s.listSignals(ctx, repoID, params)
		return map[string]any{"signals": rows}, meta, err
	case "check_architecture_rules":
		src, _ := params["source_file"].(string)
		tgt, _ := params["target_file"].(string)
		if src == "" || tgt == "" {
			return nil, meta, fmt.Errorf("source_file and target_file are required")
		}
		if s.drift == nil {
			return nil, meta, fmt.Errorf("drift engine unavailable")
		}
		violations, err := s.drift.ValidateFile(ctx, repoID, src)
		if err != nil {
			return nil, meta, err
		}
		var matched []driftdetector.Violation
		for _, v := range violations {
			if v.SourceFile == src && v.TargetFile == tgt {
				matched = append(matched, v)
			}
		}
		return map[string]any{"violations": matched, "allowed": len(matched) == 0}, meta, nil
	default:
		return nil, meta, fmt.Errorf("unknown tool: %s", tool)
	}
}

func (s *Server) fileContext(ctx context.Context, repoID int64, path string) (map[string]any, error) {
	path = strings.TrimPrefix(strings.TrimSpace(path), "./")
	var fileID int64
	var metrics socio.FileMetrics
	err := s.pool.QueryRow(ctx, `
		SELECT f.id,
		       COALESCE(fm.churn_score, 0), COALESCE(fm.risk_level, 'low'),
		       COALESCE(fm.bus_factor, 0), COALESCE(fm.is_hotspot, false)
		FROM files f
		LEFT JOIN file_metrics fm ON fm.file_id = f.id AND fm.repository_id = f.repository_id
		WHERE f.repository_id = $1 AND f.relative_path = $2
	`, repoID, path).Scan(&fileID, &metrics.ChurnScore, &metrics.RiskLevel, &metrics.BusFactor, &metrics.IsHotspot)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	var signalCount int
	_ = s.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM architecture_signals
		WHERE repository_id = $1 AND file_id = $2 AND confidence >= 0.7
	`, repoID, fileID).Scan(&signalCount)
	out := map[string]any{
		"file_path":                 path,
		"risk_level":                metrics.RiskLevel,
		"churn_score":               metrics.ChurnScore,
		"bus_factor":                metrics.BusFactor,
		"is_hotspot":                metrics.IsHotspot,
		"architecture_signal_count": signalCount,
	}
	if s.socio != nil {
		rows, _ := s.socio.GetOwnership(ctx, repoID, fileID)
		if len(rows) > 0 {
			out["ownership"] = rows[0]
		}
	}
	return out, nil
}

func (s *Server) listSignals(ctx context.Context, repoID int64, params map[string]any) ([]map[string]any, error) {
	q := `SELECT f.relative_path, s.signal_type, s.summary, s.confidence
		FROM architecture_signals s
		LEFT JOIN files f ON f.id = s.file_id
		WHERE s.repository_id = $1 AND s.confidence >= 0.7`
	args := []any{repoID}
	if fp, ok := params["file_path"].(string); ok && fp != "" {
		q += ` AND f.relative_path = $2`
		args = append(args, fp)
	}
	if st, ok := params["signal_type"].(string); ok && st != "" {
		q += fmt.Sprintf(` AND s.signal_type = $%d`, len(args)+1)
		args = append(args, st)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var path, stype, summary string
		var conf float64
		if err := rows.Scan(&path, &stype, &summary, &conf); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"file_path": path, "signal_type": stype, "summary": summary, "confidence": conf,
		})
	}
	return out, rows.Err()
}

func (s *Server) graphCompleteness(ctx context.Context, repoID int64) float64 {
	var total, indexed int
	_ = s.pool.QueryRow(ctx, `SELECT count(*)::int FROM files WHERE repository_id = $1`, repoID).Scan(&total)
	if total == 0 {
		return 0
	}
	_ = s.pool.QueryRow(ctx, `
		SELECT count(DISTINCT file_id)::int FROM symbols WHERE repository_id = $1
	`, repoID).Scan(&indexed)
	if indexed > total {
		indexed = total
	}
	return float64(indexed) / float64(total)
}

func parseRepoID(v any) (int64, error) {
	switch x := v.(type) {
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid repository_id")
		}
		return id, nil
	case float64:
		if x <= 0 {
			return 0, fmt.Errorf("invalid repository_id")
		}
		return int64(x), nil
	default:
		return 0, fmt.Errorf("repository_id is required")
	}
}

func manifest() map[string]any {
	return map[string]any{
		"name":        "codeatlas",
		"version":     "1.0.0",
		"description": "Architecture intelligence for your codebase. Answers ownership, blast radius, risk, and dependency questions.",
		"tools": []map[string]any{
			{"name": "get_file_context", "description": "Get ownership, churn, risk level, and architecture signals for a file path."},
			{"name": "get_blast_radius", "description": "Find all files that depend on a given file or function."},
			{"name": "get_ownership", "description": "Find who owns a file or module."},
			{"name": "get_hotspots", "description": "Return the highest-risk files in the repository."},
			{"name": "get_architecture_signals", "description": "Return engineering signals from PR history."},
			{"name": "check_architecture_rules", "description": "Check if a proposed change violates architecture rules."},
		},
	}
}

type logEntry struct {
	Tool         string    `json:"tool"`
	RepositoryID string    `json:"repositoryId,omitempty"`
	At           time.Time `json:"at"`
	OK           bool      `json:"ok"`
	Error        string    `json:"error,omitempty"`
}

type callLog struct {
	mu    sync.Mutex
	buf   []logEntry
	limit int
}

func newCallLog(limit int) *callLog {
	return &callLog{limit: limit}
}

func (c *callLog) add(e logEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buf = append(c.buf, e)
	if len(c.buf) > c.limit {
		c.buf = c.buf[len(c.buf)-c.limit:]
	}
}

func (c *callLog) recent(n int) []logEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n >= len(c.buf) {
		out := make([]logEntry, len(c.buf))
		copy(out, c.buf)
		return out
	}
	out := make([]logEntry, n)
	copy(out, c.buf[len(c.buf)-n:])
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

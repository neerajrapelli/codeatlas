package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"codeatlas/apps/api/internal/socio"
	"codeatlas/apps/api/internal/telemetry"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"github.com/pgvector/pgvector-go"
)

const (
	semanticSeedLimit   = 6
	maxGraphHopDepth    = 2
	maxExpandedFiles    = 42
	maxNeighborsPerFile = 24
)

type Retriever struct {
	pool  *pgxpool.Pool
	socio *socio.Store
}

func NewRetriever(pool *pgxpool.Pool, socioStore *socio.Store) *Retriever {
	return &Retriever{pool: pool, socio: socioStore}
}

func (r *Retriever) RetrieveContext(ctx context.Context, repositoryID int64, tenantID string, queryEmbedding []float32, limit int) ([]ContextItem, error) {
	ctx, span := telemetry.Start(ctx, "ai", "ai.RetrieveContext",
		attribute.Int64("repository.id", repositoryID),
		attribute.Int("context.limit", limit),
	)
	defer span.End()

	seedFileIDs, err := r.semanticFileSeeds(ctx, repositoryID, tenantID, queryEmbedding, semanticSeedLimit)
	if err != nil {
		return nil, err
	}
	expanded, err := r.expandDependencies(ctx, repositoryID, tenantID, seedFileIDs, maxExpandedFiles)
	if err != nil {
		return nil, err
	}
	items, err := r.loadContextItems(ctx, repositoryID, tenantID, expanded)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Importance > items[j].Importance })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *Retriever) semanticFileSeeds(ctx context.Context, repositoryID int64, tenantID string, queryEmbedding []float32, limit int) ([]int64, error) {
	vec := pgvector.NewVector(queryEmbedding)
	rows, err := r.pool.Query(ctx, `
		SELECT e.file_id, 1 - (e.embedding <=> $2::vector) AS score
		FROM entity_embeddings e
		WHERE e.repository_id = $1
		  AND e.file_id IS NOT NULL
		  AND ($4 = '' OR e.tenant_id = $4)
		ORDER BY e.embedding <=> $2::vector
		LIMIT $3
	`, repositoryID, vec, limit, tenantID)
	if err != nil {
		return nil, fmt.Errorf("semantic query: %w", err)
	}
	defer rows.Close()

	out := make([]int64, 0, limit)
	for rows.Next() {
		var fileID int64
		var score float64
		if err := rows.Scan(&fileID, &score); err != nil {
			return nil, fmt.Errorf("scan semantic result: %w", err)
		}
		out = append(out, fileID)
	}
	return out, rows.Err()
}

// expandDependencies performs BFS up to maxGraphHopDepth on import/export edges (bidirectional).
func (r *Retriever) expandDependencies(ctx context.Context, repositoryID int64, tenantID string, seedFileIDs []int64, limit int) ([]int64, error) {
	type node struct {
		id    int64
		depth int
	}
	seen := make(map[int64]int, len(seedFileIDs))
	queue := make([]node, 0, len(seedFileIDs))
	for _, id := range seedFileIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = 0
		queue = append(queue, node{id: id, depth: 0})
	}

	for head := 0; head < len(queue) && len(seen) < limit; head++ {
		cur := queue[head]
		if cur.depth >= maxGraphHopDepth {
			continue
		}
		neighbors, err := r.fileNeighbors(ctx, repositoryID, tenantID, cur.id)
		if err != nil {
			return nil, err
		}
		for _, next := range neighbors {
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = cur.depth + 1
			queue = append(queue, node{id: next, depth: cur.depth + 1})
			if len(seen) >= limit {
				break
			}
		}
	}

	out := make([]int64, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}

func (r *Retriever) fileNeighbors(ctx context.Context, repositoryID int64, tenantID string, fileID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT to_file_id FROM file_dependencies
		WHERE repository_id = $1 AND from_file_id = $2 AND ($3 = '' OR tenant_id = $3)
		LIMIT $4
	`, repositoryID, fileID, tenantID, maxNeighborsPerFile)
	if err != nil {
		return nil, fmt.Errorf("outgoing deps: %w", err)
	}
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, id)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx, `
		SELECT from_file_id FROM file_dependencies
		WHERE repository_id = $1 AND to_file_id = $2 AND ($3 = '' OR tenant_id = $3)
		LIMIT $4
	`, repositoryID, fileID, tenantID, maxNeighborsPerFile)
	if err != nil {
		return nil, fmt.Errorf("incoming deps: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// KnownFilePaths returns relative paths indexed for a repository (for hallucination checks).
func (r *Retriever) KnownFilePaths(ctx context.Context, repositoryID int64, tenantID string) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, relative_path FROM files
		WHERE repository_id = $1 AND ($2 = '' OR tenant_id = $2)
	`, repositoryID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		out[path] = id
	}
	return out, rows.Err()
}

// ValidateMentions checks paths and rule names against indexed repository data.
func (r *Retriever) ValidateMentions(ctx context.Context, repositoryID int64, tenantID string, paths []string, ruleNames []string) (map[string]bool, map[string]bool, error) {
	pathOK := make(map[string]bool, len(paths))
	ruleOK := make(map[string]bool, len(ruleNames))

	known, err := r.KnownFilePaths(ctx, repositoryID, tenantID)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, pathOK[p] = known[p]
	}

	if len(ruleNames) > 0 {
		rows, err := r.pool.Query(ctx, `
			SELECT name FROM architecture_rules
			WHERE repository_id = $1 AND ($2 = '' OR tenant_id = $2)
		`, repositoryID, tenantID)
		if err != nil {
			return pathOK, nil, err
		}
		defer rows.Close()
		rules := make(map[string]struct{})
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return pathOK, nil, err
			}
			rules[name] = struct{}{}
		}
		for _, n := range ruleNames {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			_, ruleOK[n] = rules[n]
		}
	}
	return pathOK, ruleOK, nil
}

func (r *Retriever) loadContextItems(ctx context.Context, repositoryID int64, tenantID string, fileIDs []int64) ([]ContextItem, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			f.id,
			f.relative_path,
			COALESCE(array_agg(DISTINCT fi.module_path) FILTER (WHERE fi.module_path IS NOT NULL), '{}') AS imports,
			COALESCE(array_agg(DISTINCT fe.export_name) FILTER (WHERE fe.export_name IS NOT NULL), '{}') AS exports,
			COALESCE(array_agg(DISTINCT s.name) FILTER (WHERE s.name IS NOT NULL), '{}') AS symbols,
			(SELECT count(*) FROM file_dependencies d WHERE d.repository_id = f.repository_id AND d.from_file_id = f.id) AS dep_out,
			(SELECT count(*) FROM file_dependencies d WHERE d.repository_id = f.repository_id AND d.to_file_id = f.id) AS dep_in
		FROM files f
		LEFT JOIN file_imports fi ON fi.file_id = f.id AND ($3 = '' OR fi.tenant_id = $3)
		LEFT JOIN file_exports fe ON fe.file_id = f.id AND ($3 = '' OR fe.tenant_id = $3)
		LEFT JOIN symbols s ON s.file_id = f.id AND ($3 = '' OR s.tenant_id = $3)
		WHERE f.repository_id = $1
		  AND f.id = ANY($2)
		  AND ($3 = '' OR f.tenant_id = $3)
		GROUP BY f.id, f.relative_path
	`, repositoryID, fileIDs, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load context files: %w", err)
	}
	defer rows.Close()

	items := make([]ContextItem, 0, len(fileIDs))
	for rows.Next() {
		var item ContextItem
		if err := rows.Scan(&item.FileID, &item.Path, &item.Imports, &item.Exports, &item.Symbols, &item.DependencyOut, &item.DependencyIn); err != nil {
			return nil, fmt.Errorf("scan context item: %w", err)
		}
		item.Imports = compact(item.Imports, 8)
		item.Exports = compact(item.Exports, 8)
		item.Symbols = compact(item.Symbols, 12)
		item.Importance = float64(item.DependencyOut*2 + item.DependencyIn)
		if strings.Contains(item.Path, "auth") {
			item.Importance += 2
		}
		item.SelectionLabel = "semantic+graph-2hop"
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if r.socio != nil && len(items) > 0 {
		ids := make([]int64, len(items))
		for i := range items {
			ids[i] = items[i].FileID
		}
		ctxMap, err := r.socio.SocioContextForFiles(ctx, repositoryID, ids)
		if err != nil {
			return items, nil
		}
		for i := range items {
			if sc, ok := ctxMap[items[i].FileID]; ok {
				items[i].DominantOwnerLogin = sc.DominantOwnerLogin
				items[i].BusFactor = sc.BusFactor
				items[i].ChurnScore = sc.ChurnScore
				items[i].RiskLevel = sc.RiskLevel
				items[i].IsHotspot = sc.IsHotspot
				items[i].HasBusFactorRisk = sc.HasBusFactorRisk
				items[i].CommitCount90d = sc.CommitCount90d
				if sc.IsHotspot || sc.HasBusFactorRisk {
					items[i].Importance += 3
				}
			}
		}
	}
	return items, nil
}

func compact(in []string, limit int) []string {
	if len(in) <= limit {
		return in
	}
	return in[:limit]
}

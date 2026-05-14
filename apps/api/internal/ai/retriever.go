package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type Retriever struct {
	pool *pgxpool.Pool
}

func NewRetriever(pool *pgxpool.Pool) *Retriever {
	return &Retriever{pool: pool}
}

func (r *Retriever) RetrieveContext(ctx context.Context, repositoryID int64, queryEmbedding []float32, limit int) ([]ContextItem, error) {
	seedFileIDs, err := r.semanticFileSeeds(ctx, repositoryID, queryEmbedding, limit)
	if err != nil {
		return nil, err
	}
	expanded, err := r.expandDependencies(ctx, repositoryID, seedFileIDs, limit*3)
	if err != nil {
		return nil, err
	}
	items, err := r.loadContextItems(ctx, repositoryID, expanded)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Importance > items[j].Importance })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *Retriever) semanticFileSeeds(ctx context.Context, repositoryID int64, queryEmbedding []float32, limit int) ([]int64, error) {
	vec := pgvector.NewVector(queryEmbedding)
	rows, err := r.pool.Query(ctx, `
		SELECT e.file_id, 1 - (e.embedding <=> $2::vector) AS score
		FROM entity_embeddings e
		WHERE e.repository_id = $1
		  AND e.file_id IS NOT NULL
		ORDER BY e.embedding <=> $2::vector
		LIMIT $3
	`, repositoryID, vec, limit)
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

func (r *Retriever) expandDependencies(ctx context.Context, repositoryID int64, seedFileIDs []int64, limit int) ([]int64, error) {
	seen := make(map[int64]struct{}, len(seedFileIDs))
	queue := make([]int64, 0, len(seedFileIDs))
	for _, id := range seedFileIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		queue = append(queue, id)
	}
	for idx := 0; idx < len(queue) && len(queue) < limit; idx++ {
		current := queue[idx]
		rows, err := r.pool.Query(ctx, `
			SELECT to_file_id
			FROM file_dependencies
			WHERE repository_id = $1 AND from_file_id = $2
			LIMIT 20
		`, repositoryID, current)
		if err != nil {
			return nil, fmt.Errorf("dependency expansion: %w", err)
		}
		for rows.Next() {
			var next int64
			if err := rows.Scan(&next); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan dependency expansion: %w", err)
			}
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			queue = append(queue, next)
			if len(queue) >= limit {
				break
			}
		}
		rows.Close()
	}
	return queue, nil
}

func (r *Retriever) loadContextItems(ctx context.Context, repositoryID int64, fileIDs []int64) ([]ContextItem, error) {
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
		LEFT JOIN file_imports fi ON fi.file_id = f.id
		LEFT JOIN file_exports fe ON fe.file_id = f.id
		LEFT JOIN symbols s ON s.file_id = f.id
		WHERE f.repository_id = $1
		  AND f.id = ANY($2)
		GROUP BY f.id, f.relative_path
	`, repositoryID, fileIDs)
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
		item.SelectionLabel = "semantic+graph"
		items = append(items, item)
	}
	return items, rows.Err()
}

func compact(in []string, limit int) []string {
	if len(in) <= limit {
		return in
	}
	return in[:limit]
}

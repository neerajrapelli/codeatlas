package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type graphFile struct {
	ID      string           `json:"id"`
	Path    string           `json:"path"`
	Imports []string         `json:"imports"`
	Exports []string         `json:"exports"`
	Symbols []map[string]any `json:"symbols"`
}

type graphDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type graphPage struct {
	Limit  int
	Offset int
}

func parseGraphPage(r *http.Request, defaultLimit, maxLimit int) graphPage {
	limit := defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return graphPage{Limit: limit, Offset: offset}
}

func parseGraphDepth(r *http.Request, defaultDepth, maxDepth int) int {
	depth := defaultDepth
	if v := r.URL.Query().Get("depth"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			depth = n
		}
	}
	if depth > maxDepth {
		depth = maxDepth
	}
	return depth
}

func loadGraphPayload(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, tenantID string, page graphPage) (files []graphFile, deps []graphDependency, total int, truncated bool, err error) {
	if err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM files f
		WHERE f.repository_id = $1 AND ($2 = '' OR f.tenant_id = $2)
	`, repositoryID, tenantID).Scan(&total); err != nil {
		return nil, nil, 0, false, err
	}

	fileRows, err := pool.Query(ctx, `
		SELECT f.id, f.relative_path
		FROM files f
		WHERE f.repository_id = $1 AND ($2 = '' OR f.tenant_id = $2)
		ORDER BY f.relative_path
		LIMIT $3 OFFSET $4
	`, repositoryID, tenantID, page.Limit, page.Offset)
	if err != nil {
		return nil, nil, 0, false, err
	}
	defer fileRows.Close()

	files = make([]graphFile, 0, page.Limit)
	fileIDs := make([]int64, 0, page.Limit)
	fileIDToNode := make(map[int64]string, page.Limit)
	for fileRows.Next() {
		var id int64
		var path string
		if err := fileRows.Scan(&id, &path); err != nil {
			return nil, nil, 0, false, err
		}
		nodeID := strconv.FormatInt(id, 10)
		fileIDs = append(fileIDs, id)
		fileIDToNode[id] = nodeID
		files = append(files, graphFile{
			ID:      nodeID,
			Path:    path,
			Imports: []string{},
			Exports: []string{},
			Symbols: []map[string]any{},
		})
	}
	if err := fileRows.Err(); err != nil {
		return nil, nil, 0, false, err
	}
	if len(fileIDs) == 0 {
		return files, nil, total, page.Offset+len(files) < total, nil
	}

	if err := attachImportsExports(ctx, pool, repositoryID, tenantID, fileIDs, fileIDToNode, files); err != nil {
		return nil, nil, 0, false, err
	}
	if err := attachSymbols(ctx, pool, repositoryID, tenantID, fileIDs, fileIDToNode, files); err != nil {
		return nil, nil, 0, false, err
	}
	deps, err = loadDependenciesForFiles(ctx, pool, repositoryID, tenantID, fileIDToNode)
	if err != nil {
		return nil, nil, 0, false, err
	}
	truncated = page.Offset+len(files) < total
	return files, deps, total, truncated, nil
}

func attachImportsExports(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, tenantID string, fileIDs []int64, fileIDToNode map[int64]string, files []graphFile) error {
	rows, err := pool.Query(ctx, `
		SELECT file_id, module_path FROM file_imports
		WHERE repository_id = $1 AND ($2 = '' OR tenant_id = $2) AND file_id = ANY($3)
	`, repositoryID, tenantID, fileIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	importsByID := map[int64][]string{}
	for rows.Next() {
		var fileID int64
		var mod string
		if err := rows.Scan(&fileID, &mod); err != nil {
			return err
		}
		importsByID[fileID] = append(importsByID[fileID], mod)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	erows, err := pool.Query(ctx, `
		SELECT file_id, export_name FROM file_exports
		WHERE repository_id = $1 AND ($2 = '' OR tenant_id = $2) AND file_id = ANY($3)
	`, repositoryID, tenantID, fileIDs)
	if err != nil {
		return err
	}
	defer erows.Close()
	exportsByID := map[int64][]string{}
	for erows.Next() {
		var fileID int64
		var name string
		if err := erows.Scan(&fileID, &name); err != nil {
			return err
		}
		exportsByID[fileID] = append(exportsByID[fileID], name)
	}
	if err := erows.Err(); err != nil {
		return err
	}

	for i := range files {
		id, _ := strconv.ParseInt(files[i].ID, 10, 64)
		files[i].Imports = importsByID[id]
		files[i].Exports = exportsByID[id]
	}
	_ = fileIDToNode
	return nil
}

func attachSymbols(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, tenantID string, fileIDs []int64, fileIDToNode map[int64]string, files []graphFile) error {
	rows, err := pool.Query(ctx, `
		SELECT file_id, name, kind FROM symbols
		WHERE repository_id = $1 AND ($2 = '' OR tenant_id = $2) AND file_id = ANY($3)
	`, repositoryID, tenantID, fileIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	symbolsByNode := map[string][]map[string]any{}
	for rows.Next() {
		var fileID int64
		var name, kind string
		if err := rows.Scan(&fileID, &name, &kind); err != nil {
			return err
		}
		nodeID, ok := fileIDToNode[fileID]
		if !ok {
			continue
		}
		symbolsByNode[nodeID] = append(symbolsByNode[nodeID], map[string]any{"name": name, "kind": kind})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range files {
		files[i].Symbols = symbolsByNode[files[i].ID]
	}
	return nil
}

func loadDependenciesForFiles(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, tenantID string, fileIDToNode map[int64]string) ([]graphDependency, error) {
	ids := make([]int64, 0, len(fileIDToNode))
	for id := range fileIDToNode {
		ids = append(ids, id)
	}
	rows, err := pool.Query(ctx, `
		SELECT from_file_id, to_file_id FROM file_dependencies
		WHERE repository_id = $1 AND ($2 = '' OR tenant_id = $2)
		  AND from_file_id = ANY($3) AND to_file_id = ANY($3)
	`, repositoryID, tenantID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deps := make([]graphDependency, 0, len(ids)*2)
	for rows.Next() {
		var fromID, toID int64
		if err := rows.Scan(&fromID, &toID); err != nil {
			return nil, err
		}
		fromNode, okFrom := fileIDToNode[fromID]
		toNode, okTo := fileIDToNode[toID]
		if !okFrom || !okTo {
			continue
		}
		deps = append(deps, graphDependency{From: fromNode, To: toNode})
	}
	return deps, rows.Err()
}

func loadSingleGraphFile(ctx context.Context, pool *pgxpool.Pool, repositoryID, fileID int64, tenantID string) (graphFile, error) {
	var id int64
	var path string
	err := pool.QueryRow(ctx, `
		SELECT f.id, f.relative_path
		FROM files f
		WHERE f.repository_id = $1 AND f.id = $2 AND ($3 = '' OR f.tenant_id = $3)
	`, repositoryID, fileID, tenantID).Scan(&id, &path)
	if err != nil {
		return graphFile{}, err
	}
	nodeID := strconv.FormatInt(id, 10)
	gf := graphFile{ID: nodeID, Path: path, Imports: []string{}, Exports: []string{}, Symbols: []map[string]any{}}
	fileIDToNode := map[int64]string{id: nodeID}
	files := []graphFile{gf}
	if err := attachImportsExports(ctx, pool, repositoryID, tenantID, []int64{id}, fileIDToNode, files); err != nil {
		return graphFile{}, err
	}
	if err := attachSymbols(ctx, pool, repositoryID, tenantID, []int64{id}, fileIDToNode, files); err != nil {
		return graphFile{}, err
	}
	return files[0], nil
}

type symbolRow struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func loadSymbolsForFile(ctx context.Context, pool *pgxpool.Pool, repositoryID, fileID int64, tenantID string) ([]symbolRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT name, kind FROM symbols
		WHERE repository_id = $1 AND file_id = $2 AND ($3 = '' OR tenant_id = $3)
		ORDER BY name
	`, repositoryID, fileID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query symbols: %w", err)
	}
	defer rows.Close()
	out := make([]symbolRow, 0, 64)
	for rows.Next() {
		var s symbolRow
		if err := rows.Scan(&s.Name, &s.Kind); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

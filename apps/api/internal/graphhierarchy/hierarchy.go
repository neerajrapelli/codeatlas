package graphhierarchy

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Layer models one hierarchical view under a path prefix (folder navigation).
type Layer struct {
	Prefix   string           `json:"prefix"`
	Clusters []ClusterSummary `json:"clusters"`
	Files    []FileSummary    `json:"files"`
	Edges    []AggEdge        `json:"edges"`
}

type ClusterSummary struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	PathPrefix     string  `json:"pathPrefix"`
	Level          int     `json:"level"`
	FileCount      int     `json:"fileCount"`
	InternalEdges  int     `json:"internalEdges"`
	Density        float64 `json:"density"`
	HasChildren    bool    `json:"hasChildren"`
}

type FileSummary struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	SymbolCount int    `json:"symbolCount"`
}

type AggEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Count int    `json:"count"`
}

type fileRow struct {
	id       int64
	path     string
	symCount int
}

// BuildLayer returns clusters + direct files under prefix, plus aggregated cross-node edges.
func BuildLayer(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, prefix string) (*Layer, error) {
	prefix = normPath(prefix)

	files, deps, err := loadRepoGraph(ctx, pool, repositoryID)
	if err != nil {
		return nil, err
	}

	pathByID := make(map[int64]string, len(files))
	for _, f := range files {
		pathByID[f.id] = normPath(f.path)
	}

	// Collect immediate children: cluster prefixes and file leaves.
	type agg struct {
		pathPrefix string
		label      string
		fileIDs    []int64
	}
	clusters := map[string]*agg{}
	directFiles := []fileRow{}

	for _, f := range files {
		p := normPath(f.path)
		kind, cp, fid := childSlot(p, prefix, f.id)
		switch kind {
		case "cluster":
			a := clusters[cp]
			if a == nil {
				a = &agg{
					pathPrefix: cp,
					label:      clusterLabel(cp, prefix),
					fileIDs:    nil,
				}
				clusters[cp] = a
			}
			a.fileIDs = append(a.fileIDs, f.id)
		case "file":
			if fid != f.id {
				continue
			}
			directFiles = append(directFiles, f)
		case "":
			continue
		}
	}

	// Edge aggregation at this abstraction level.
	edgeCount := map[string]map[string]int{}
	for _, d := range deps {
		fromPath, ok1 := pathByID[d.from]
		toPath, ok2 := pathByID[d.to]
		if !ok1 || !ok2 {
			continue
		}
		a := NodeKey(repositoryID, fromPath, prefix, d.from)
		b := NodeKey(repositoryID, toPath, prefix, d.to)
		if a == "" || b == "" || a == b {
			continue
		}
		if edgeCount[a] == nil {
			edgeCount[a] = map[string]int{}
		}
		edgeCount[a][b]++
	}

	edges := make([]AggEdge, 0)
	for from, tos := range edgeCount {
		for to, c := range tos {
			if c <= 0 {
				continue
			}
			edges = append(edges, AggEdge{From: from, To: to, Count: c})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Count != edges[j].Count {
			return edges[i].Count > edges[j].Count
		}
		return edges[i].From < edges[j].From
	})

	prefixDepth := 0
	if prefix != "" {
		prefixDepth = strings.Count(prefix, "/") + 1
	}

	outClusters := make([]ClusterSummary, 0, len(clusters))
	for cp, a := range clusters {
		fc := countFilesUnderPrefix(files, cp)
		internal := countInternalEdges(deps, pathByID, cp)
		density := 0.0
		if fc > 1 {
			density = float64(internal) / float64(fc*fc)
		}
		hasChildren := hasClusterChildren(files, cp)
		outClusters = append(outClusters, ClusterSummary{
			ID:            clusterNodeID(repositoryID, cp),
			Label:         a.label,
			PathPrefix:    cp,
			Level:         prefixDepth + 1,
			FileCount:     fc,
			InternalEdges: internal,
			Density:       density,
			HasChildren:   hasChildren,
		})
	}
	sort.Slice(outClusters, func(i, j int) bool {
		return outClusters[i].Label < outClusters[j].Label
	})

	outFiles := make([]FileSummary, 0, len(directFiles))
	for _, f := range directFiles {
		outFiles = append(outFiles, FileSummary{
			ID:          strconv.FormatInt(f.id, 10),
			Path:        f.path,
			SymbolCount: f.symCount,
		})
	}
	sort.Slice(outFiles, func(i, j int) bool {
		return outFiles[i].Path < outFiles[j].Path
	})

	return &Layer{
		Prefix:   prefix,
		Clusters: outClusters,
		Files:    outFiles,
		Edges:    edges,
	}, nil
}

func loadRepoGraph(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) ([]fileRow, []struct{ from, to int64 }, error) {
	rows, err := pool.Query(ctx, `
		SELECT f.id, f.relative_path,
		       COALESCE((SELECT COUNT(*) FROM symbols s WHERE s.file_id = f.id AND s.repository_id = f.repository_id), 0)::int
		FROM files f
		WHERE f.repository_id = $1
		ORDER BY f.relative_path
	`, repositoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	var files []fileRow
	for rows.Next() {
		var fr fileRow
		if err := rows.Scan(&fr.id, &fr.path, &fr.symCount); err != nil {
			return nil, nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	drows, err := pool.Query(ctx, `
		SELECT from_file_id, to_file_id FROM file_dependencies WHERE repository_id = $1
	`, repositoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("list deps: %w", err)
	}
	defer drows.Close()

	var deps []struct{ from, to int64 }
	for drows.Next() {
		var from, to int64
		if err := drows.Scan(&from, &to); err != nil {
			return nil, nil, fmt.Errorf("scan dep: %w", err)
		}
		deps = append(deps, struct{ from, to int64 }{from, to})
	}
	if err := drows.Err(); err != nil {
		return nil, nil, err
	}
	return files, deps, nil
}

func normPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return p
}

// childSlot classifies a file under parentPrefix into next cluster or direct file child.
func childSlot(path, parentPrefix string, fileID int64) (kind string, clusterPath string, id int64) {
	path = normPath(path)
	parentPrefix = normPath(parentPrefix)
	var rest string
	if parentPrefix == "" {
		rest = path
	} else {
		if path == parentPrefix {
			return "", "", 0
		}
		if !strings.HasPrefix(path, parentPrefix+"/") {
			return "", "", 0
		}
		rest = strings.TrimPrefix(path, parentPrefix+"/")
	}
	if rest == "" {
		return "", "", 0
	}
	if !strings.Contains(rest, "/") {
		return "file", "", fileID
	}
	seg := strings.SplitN(rest, "/", 2)[0]
	var cp string
	if parentPrefix == "" {
		cp = seg
	} else {
		cp = parentPrefix + "/" + seg
	}
	return "cluster", cp, 0
}

func clusterLabel(clusterPath, parentPrefix string) string {
	clusterPath = normPath(clusterPath)
	parentPrefix = normPath(parentPrefix)
	if parentPrefix == "" {
		return clusterPath
	}
	return strings.TrimPrefix(clusterPath, parentPrefix+"/")
}

func clusterNodeID(repoID int64, pathPrefix string) string {
	return fmt.Sprintf("c:%d:%s", repoID, pathPrefix)
}

func fileNodeID(fileID int64) string {
	return fmt.Sprintf("f:%d", fileID)
}

// NodeKey resolves the visible graph node id for a file at the current prefix layer.
func NodeKey(repoID int64, path string, prefix string, fileID int64) string {
	path = normPath(path)
	prefix = normPath(prefix)
	kind, cp, fid := childSlot(path, prefix, fileID)
	switch kind {
	case "cluster":
		return clusterNodeID(repoID, cp)
	case "file":
		return fileNodeID(fid)
	default:
		return ""
	}
}

func countFilesUnderPrefix(files []fileRow, prefix string) int {
	prefix = normPath(prefix)
	n := 0
	for _, f := range files {
		p := normPath(f.path)
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			n++
		}
	}
	return n
}

func hasClusterChildren(files []fileRow, prefix string) bool {
	prefix = normPath(prefix)
	for _, f := range files {
		p := normPath(f.path)
		if p == prefix || !strings.HasPrefix(p, prefix+"/") {
			continue
		}
		rest := strings.TrimPrefix(p, prefix+"/")
		if strings.Contains(rest, "/") {
			return true
		}
	}
	return false
}

func countInternalEdges(deps []struct{ from, to int64 }, pathByID map[int64]string, prefix string) int {
	prefix = normPath(prefix)
	n := 0
	for _, d := range deps {
		a, ok1 := pathByID[d.from]
		b, ok2 := pathByID[d.to]
		if !ok1 || !ok2 {
			continue
		}
		a = normPath(a)
		b = normPath(b)
		if (a == prefix || strings.HasPrefix(a, prefix+"/")) &&
			(b == prefix || strings.HasPrefix(b, prefix+"/")) {
			n++
		}
	}
	return n
}

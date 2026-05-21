package graphhierarchy

// FileNode is a file vertex used when building import dependency edges.
type FileNode struct {
	ID      string
	Path    string
	Imports []string
}

// DependencyEdge connects two files in the dependency graph.
type DependencyEdge struct {
	Source string
	Target string
	Type   string
}

// BuildDependencyEdges creates import edges between files, deduplicated by source/target pair.
func BuildDependencyEdges(files []FileNode) []DependencyEdge {
	pathToID := make(map[string]string, len(files)*2)
	for _, f := range files {
		pathToID[f.Path] = f.ID
	}

	seen := make(map[string]struct{})
	edges := make([]DependencyEdge, 0)

	for _, f := range files {
		for _, imp := range f.Imports {
			targetID, ok := pathToID[imp]
			if !ok {
				continue
			}
			key := f.ID + "->" + targetID
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			edges = append(edges, DependencyEdge{
				Source: f.ID,
				Target: targetID,
				Type:   "imports",
			})
		}
	}
	return edges
}

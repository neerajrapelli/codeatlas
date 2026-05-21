package graphhierarchy

import (
	"testing"
)

func TestGraphBuildsImportEdges(t *testing.T) {
	files := []FileNode{
		{ID: "file-a", Path: "src/auth/index.ts", Imports: []string{"src/utils/crypto.ts"}},
		{ID: "file-b", Path: "src/utils/crypto.ts", Imports: []string{}},
	}

	edges := BuildDependencyEdges(files)

	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Source != "file-a" || edges[0].Target != "file-b" || edges[0].Type != "imports" {
		t.Fatalf("unexpected edge: %+v", edges[0])
	}
}

func TestGraphDeduplicatesEdges(t *testing.T) {
	files := []FileNode{
		{ID: "file-a", Path: "src/a.ts", Imports: []string{"src/b.ts", "src/b.ts"}},
		{ID: "file-b", Path: "src/b.ts", Imports: []string{}},
	}
	edges := BuildDependencyEdges(files)
	if len(edges) != 1 {
		t.Fatalf("expected 1 deduplicated edge, got %d", len(edges))
	}
}

func TestGraphHandlesCircularDependency(t *testing.T) {
	files := []FileNode{
		{ID: "file-a", Path: "src/a.ts", Imports: []string{"src/b.ts"}},
		{ID: "file-b", Path: "src/b.ts", Imports: []string{"src/a.ts"}},
	}
	func() { BuildDependencyEdges(files) }()
}

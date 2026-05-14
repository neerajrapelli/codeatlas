package indexer

import "testing"

func TestResolveLocalImport(t *testing.T) {
	repo := "/tmp/repo"
	known := map[string]struct{}{
		"src/lib/util.ts": {},
		"src/models/index.ts": {},
	}

	if got, ok := resolveLocalImport(repo, "src/main.ts", "./lib/util", known); !ok || got != "src/lib/util.ts" {
		t.Fatalf("expected src/lib/util.ts, got %q ok=%v", got, ok)
	}
	if got, ok := resolveLocalImport(repo, "src/main.ts", "./models", known); !ok || got != "src/models/index.ts" {
		t.Fatalf("expected src/models/index.ts, got %q ok=%v", got, ok)
	}
	if _, ok := resolveLocalImport(repo, "src/main.ts", "./missing", known); ok {
		t.Fatalf("expected missing import to be unresolved")
	}
}

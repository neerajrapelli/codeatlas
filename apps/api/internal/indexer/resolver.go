package indexer

import (
	"path/filepath"
	"strings"
)

func resolveDependencies(repoPath string, file ScannedFile, imports []Import, known map[string]struct{}) []string {
	deps := make([]string, 0)
	for _, imp := range imports {
		if !strings.HasPrefix(imp.ModulePath, ".") {
			continue
		}
		resolved, ok := resolveLocalImport(repoPath, file.RelativePath, imp.ModulePath, known)
		if ok {
			deps = append(deps, resolved)
		}
	}
	return deps
}

func resolveLocalImport(repoPath, fromRelPath, spec string, known map[string]struct{}) (string, bool) {
	base := filepath.Dir(filepath.Join(repoPath, filepath.FromSlash(fromRelPath)))
	candidate := filepath.Join(base, filepath.FromSlash(spec))

	check := func(path string) (string, bool) {
		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return "", false
		}
		canon := filepath.ToSlash(rel)
		_, ok := known[canon]
		return canon, ok
	}

	if rel, ok := check(candidate); ok {
		return rel, true
	}
	for _, ext := range []string{".ts", ".tsx", ".mts", ".cts"} {
		if rel, ok := check(candidate + ext); ok {
			return rel, true
		}
	}
	for _, idx := range []string{"index.ts", "index.tsx", "index.mts", "index.cts"} {
		if rel, ok := check(filepath.Join(candidate, idx)); ok {
			return rel, true
		}
	}
	return "", false
}

package indexer

import (
	"path/filepath"
	"strings"
)

var resolveExtensions = []string{
	".ts", ".tsx", ".mts", ".cts",
	".js", ".jsx", ".mjs", ".cjs",
	".py",
	".go",
	".java",
	".c", ".h",
	".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx",
	".php",
	".cs",
}

var resolveIndexFiles = []string{
	"index.ts", "index.tsx", "index.js", "index.jsx",
	"__init__.py", "mod.go",
}

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
	for _, ext := range resolveExtensions {
		if rel, ok := check(candidate + ext); ok {
			return rel, true
		}
	}
	for _, idx := range resolveIndexFiles {
		if rel, ok := check(filepath.Join(candidate, idx)); ok {
			return rel, true
		}
	}
	return "", false
}

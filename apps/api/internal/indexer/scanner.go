package indexer

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var ignoredDirs = map[string]struct{}{
	"node_modules": {},
	"dist":         {},
	"build":        {},
	".git":         {},
}

type TypeScriptFileScanner struct{}

func NewTypeScriptFileScanner() *TypeScriptFileScanner { return &TypeScriptFileScanner{} }

func (s *TypeScriptFileScanner) Scan(repoPath string) ([]ScannedFile, error) {
	files := make([]ScannedFile, 0, 1024)
	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if _, ignore := ignoredDirs[d.Name()]; ignore {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".ts" && ext != ".tsx" && ext != ".mts" && ext != ".cts" {
			return nil
		}

		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}
		files = append(files, ScannedFile{AbsolutePath: path, RelativePath: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

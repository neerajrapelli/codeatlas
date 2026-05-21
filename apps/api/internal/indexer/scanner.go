package indexer

import (
	"io/fs"
	"path/filepath"
)

var ignoredDirs = map[string]struct{}{
	"node_modules": {},
	"dist":         {},
	"build":        {},
	".git":         {},
	"vendor":       {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
}

// MultiLanguageScanner indexes source files for all supported languages.
type MultiLanguageScanner struct{}

func NewMultiLanguageScanner() *MultiLanguageScanner { return &MultiLanguageScanner{} }

// NewTypeScriptFileScanner is a backward-compatible alias.
func NewTypeScriptFileScanner() *MultiLanguageScanner { return NewMultiLanguageScanner() }

func (s *MultiLanguageScanner) Scan(repoPath string) ([]ScannedFile, error) {
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

		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		lang, ok := LanguageForPath(rel)
		if !ok {
			return nil
		}

		files = append(files, ScannedFile{
			AbsolutePath: path,
			RelativePath: rel,
			Language:     lang,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

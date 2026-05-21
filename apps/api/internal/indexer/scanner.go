package indexer

import (
	"fmt"
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
type MultiLanguageScanner struct {
	MaxFileBytes int64
	MaxFiles     int
	MaxRepoBytes int64
}

func NewMultiLanguageScanner() *MultiLanguageScanner { return &MultiLanguageScanner{} }

// NewTypeScriptFileScanner is a backward-compatible alias.
func NewTypeScriptFileScanner() *MultiLanguageScanner { return NewMultiLanguageScanner() }

func (s *MultiLanguageScanner) Scan(repoPath string) ([]ScannedFile, error) {
	files := make([]ScannedFile, 0, 1024)
	var totalBytes int64
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
		if s.MaxFiles > 0 && len(files) >= s.MaxFiles {
			return fmt.Errorf("repository exceeds max indexed files (%d)", s.MaxFiles)
		}
		if s.MaxFileBytes > 0 {
			info, statErr := d.Info()
			if statErr == nil && info.Size() > s.MaxFileBytes {
				return nil
			}
			if statErr == nil {
				totalBytes += info.Size()
				if s.MaxRepoBytes > 0 && totalBytes > s.MaxRepoBytes {
					return fmt.Errorf("repository exceeds max total size (%d bytes)", s.MaxRepoBytes)
				}
			}
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

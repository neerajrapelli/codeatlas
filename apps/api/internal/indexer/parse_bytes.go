package indexer

import (
	"fmt"
	"os"
	"path/filepath"
)

// ParseTypeScript parses TypeScript source from memory (writes a temp file for Tree-sitter).
func ParseTypeScript(source []byte) (ParsedFile, error) {
	dir, err := os.MkdirTemp("", "codeatlas-parse-*")
	if err != nil {
		return ParsedFile{}, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "input.ts")
	if err := os.WriteFile(file, source, 0o644); err != nil {
		return ParsedFile{}, fmt.Errorf("write temp: %w", err)
	}

	lang, _ := LanguageForPath("input.ts")
	parser := NewTreeSitterParser()
	return parser.Parse(ScannedFile{AbsolutePath: file, RelativePath: "input.ts", Language: lang})
}

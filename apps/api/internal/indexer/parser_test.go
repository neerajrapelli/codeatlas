package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParserExtractsImportsExportsAndSymbols(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.ts")
	source := `
import fs from 'node:fs';
import { helper } from './helper';
export interface User { id: string }
export class Service {}
function localFn() {}
export function exportedFn() {}
`
	if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	parser := NewTreeSitterTypeScriptParser()
	parsed, err := parser.Parse(ScannedFile{AbsolutePath: file, RelativePath: "sample.ts"})
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if len(parsed.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(parsed.Imports))
	}
	if len(parsed.Symbols) < 3 {
		t.Fatalf("expected at least 3 symbols, got %d", len(parsed.Symbols))
	}
	if len(parsed.Exports) < 3 {
		t.Fatalf("expected at least 3 exports, got %d", len(parsed.Exports))
	}
}

package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func importPaths(imports []Import) []string {
	out := make([]string, 0, len(imports))
	for _, imp := range imports {
		out = append(out, imp.ModulePath)
	}
	return out
}

func exportNames(exports []Export) []string {
	out := make([]string, 0, len(exports))
	for _, ex := range exports {
		out = append(out, ex.Name)
	}
	return out
}

func TestParserExtractsImports(t *testing.T) {
	src := `
        import { AuthService } from './auth/service'
        import type { User } from '../types/user'
        import * as crypto from 'crypto'
    `
	result, err := ParseTypeScript([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	paths := importPaths(result.Imports)
	if len(paths) != 3 {
		t.Fatalf("expected 3 imports, got %d (%v)", len(paths), paths)
	}
	for _, want := range []string{"./auth/service", "../types/user", "crypto"} {
		found := false
		for _, p := range paths {
			if strings.Contains(p, want) || p == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing import %q in %v", want, paths)
		}
	}
}

func TestParserExtractsExportedSymbols(t *testing.T) {
	src := `
        export class UserService { }
        export function validateToken(token: string): boolean { return true }
        export interface PublicAPI { }
        const internalHelper = () => {}
    `
	result, err := ParseTypeScript([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	names := exportNames(result.Exports)
	if len(names) < 3 {
		t.Fatalf("expected at least 3 exports, got %v", names)
	}
	for _, bad := range []string{"internalHelper"} {
		for _, n := range names {
			if n == bad {
				t.Fatalf("should not export %q", bad)
			}
		}
	}
}

func TestParserHandlesEmptyFile(t *testing.T) {
	result, err := ParseTypeScript([]byte(""))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Imports) != 0 || len(result.Exports) != 0 {
		t.Fatalf("expected empty imports/exports")
	}
}

func TestParserHandlesMalformedTypeScript(t *testing.T) {
	src := `this is not valid typescript }{{{`
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parser panicked: %v", r)
		}
	}()
	result, err := ParseTypeScript([]byte(src))
	_ = result
	_ = err
}

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

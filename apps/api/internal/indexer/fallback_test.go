package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoFallbackExtractsExports(t *testing.T) {
	src := `package main

import "fmt"

func Exported() {}
func unexported() {}
type ExportedType struct{}
`
	file := ScannedFile{RelativePath: "main.go", Language: LangGo}
	out := parseGoFallback(file, src)
	if len(out.Imports) != 1 || out.Imports[0].ModulePath != "fmt" {
		t.Fatalf("imports: %+v", out.Imports)
	}
	names := exportNames(out.Exports)
	for _, want := range []string{"Exported", "ExportedType"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing export %q in %v", want, names)
		}
	}
	for _, n := range names {
		if n == "unexported" {
			t.Fatal("should not export unexported")
		}
	}
}

func TestParsePythonFallbackExtractsSymbols(t *testing.T) {
	src := `from os import path
import json

def public_fn():
    pass

class Widget:
    pass
`
	file := ScannedFile{RelativePath: "app.py", Language: LangPython}
	out := parsePythonFallback(file, src)
	if len(out.Imports) < 2 {
		t.Fatalf("expected imports, got %+v", out.Imports)
	}
	hasFn, hasClass := false, false
	for _, sym := range out.Symbols {
		if sym.Name == "public_fn" && sym.Kind == SymbolFunction {
			hasFn = true
		}
		if sym.Name == "Widget" && sym.Kind == SymbolClass {
			hasClass = true
		}
	}
	if !hasFn || !hasClass {
		t.Fatalf("symbols: %+v", out.Symbols)
	}
}

func TestMultiLanguageScannerFindsExtensions(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"app.ts":      "export const x = 1",
		"main.go":     "package main",
		"lib.py":      "x = 1",
		"Main.java":   "class Main {}",
		"util.cpp":    "void f() {}",
		"index.php":   "<?php",
		"Program.cs":  "class P {}",
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	scanned, err := NewMultiLanguageScanner().Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanned) != len(files) {
		t.Fatalf("got %d files, want %d: %+v", len(scanned), len(files), scanned)
	}
}

//go:build !cgo

package indexer

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	importRE    = regexp.MustCompile(`(?m)^\s*import(?:\s+type)?[\s\w{},*\n\r]*from\s+["']([^"']+)["']`)
	importBare  = regexp.MustCompile(`(?m)^\s*import\s+["']([^"']+)["']`)
	functionRE  = regexp.MustCompile(`(?m)^\s*(export\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)`)
	classRE     = regexp.MustCompile(`(?m)^\s*(export\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	interfaceRE = regexp.MustCompile(`(?m)^\s*(export\s+)?interface\s+([A-Za-z_][A-Za-z0-9_]*)`)
	exportRE    = regexp.MustCompile(`(?m)^\s*export\s+\{([^}]+)\}`)
)

type TreeSitterTypeScriptParser struct{}

func NewTreeSitterTypeScriptParser() *TreeSitterTypeScriptParser {
	return &TreeSitterTypeScriptParser{}
}

func (p *TreeSitterTypeScriptParser) Parse(file ScannedFile) (ParsedFile, error) {
	sourceBytes, err := os.ReadFile(file.AbsolutePath)
	if err != nil {
		return ParsedFile{}, fmt.Errorf("read source %s: %w", file.RelativePath, err)
	}
	source := string(sourceBytes)
	out := ParsedFile{File: file}

	for _, match := range importRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{
			ModulePath: match[1],
			TypeOnly:   strings.Contains(match[0], "import type"),
		})
	}
	for _, match := range importBare.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}

	appendSymbol := func(kind SymbolKind, exported bool, name string) {
		sym := Symbol{Name: name, Kind: kind, Exported: exported}
		out.Symbols = append(out.Symbols, sym)
		if exported {
			out.Exports = append(out.Exports, Export{Name: name})
		}
	}

	for _, match := range functionRE.FindAllStringSubmatch(source, -1) {
		appendSymbol(SymbolFunction, strings.TrimSpace(match[1]) != "", match[2])
	}
	for _, match := range classRE.FindAllStringSubmatch(source, -1) {
		appendSymbol(SymbolClass, strings.TrimSpace(match[1]) != "", match[2])
	}
	for _, match := range interfaceRE.FindAllStringSubmatch(source, -1) {
		appendSymbol(SymbolInterface, strings.TrimSpace(match[1]) != "", match[2])
	}

	for _, match := range exportRE.FindAllStringSubmatch(source, -1) {
		parts := strings.Split(match[1], ",")
		for _, part := range parts {
			name := strings.TrimSpace(strings.Split(part, " as ")[0])
			if name != "" {
				out.Exports = append(out.Exports, Export{Name: name})
			}
		}
	}

	return out, nil
}

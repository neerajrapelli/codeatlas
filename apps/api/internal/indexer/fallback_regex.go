//go:build !cgo

package indexer

import (
	"regexp"
	"strings"
)

var (
	importRE       = regexp.MustCompile(`(?m)^\s*import(?:\s+type)?[\s\w{},*\n\r]*from\s+["']([^"']+)["']`)
	importBareRE   = regexp.MustCompile(`(?m)^\s*import\s+["']([^"']+)["']`)
	functionRE     = regexp.MustCompile(`(?m)^\s*(export\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)`)
	classRE        = regexp.MustCompile(`(?m)^\s*(export\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	interfaceRE    = regexp.MustCompile(`(?m)^\s*(export\s+)?interface\s+([A-Za-z_][A-Za-z0-9_]*)`)
	exportRE       = regexp.MustCompile(`(?m)^\s*export\s+\{([^}]+)\}`)
	goImportRE     = regexp.MustCompile(`(?m)^\s*import\s+(?:\w+\s+)?["']([^"']+)["']`)
	goFuncRE       = regexp.MustCompile(`(?m)^\s*func\s+([A-Z][A-Za-z0-9_]*)\s*\(`)
	goTypeRE       = regexp.MustCompile(`(?m)^\s*type\s+([A-Z][A-Za-z0-9_]*)\s+`)
	pyImportRE     = regexp.MustCompile(`(?m)^\s*(?:from\s+([a-zA-Z0-9_.]+)\s+import|import\s+([a-zA-Z0-9_.]+))`)
	pyDefRE        = regexp.MustCompile(`(?m)^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	pyClassRE      = regexp.MustCompile(`(?m)^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\s*`)
	javaImportRE   = regexp.MustCompile(`(?m)^\s*import\s+([a-zA-Z0-9_.]+)\s*;`)
	javaClassRE    = regexp.MustCompile(`(?m)^\s*(?:public\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	cIncludeRE     = regexp.MustCompile(`(?m)^\s*#\s*include\s+"([^"]+)"`)
	phpUseRE       = regexp.MustCompile(`(?m)^\s*use\s+([A-Za-z0-9\\]+)\s*;`)
	csharpUsingRE  = regexp.MustCompile(`(?m)^\s*using\s+([A-Za-z0-9_.]+)\s*;`)
	csharpClassRE  = regexp.MustCompile(`(?m)^\s*(?:public\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

func parseTSJSFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range importRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1], TypeOnly: strings.Contains(match[0], "import type")})
	}
	for _, match := range importBareRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}
	appendSym := func(kind SymbolKind, exported bool, name string) {
		out.Symbols = append(out.Symbols, Symbol{Name: name, Kind: kind, Exported: exported})
		if exported {
			out.Exports = append(out.Exports, Export{Name: name})
		}
	}
	for _, match := range functionRE.FindAllStringSubmatch(source, -1) {
		appendSym(SymbolFunction, strings.TrimSpace(match[1]) != "", match[2])
	}
	for _, match := range classRE.FindAllStringSubmatch(source, -1) {
		appendSym(SymbolClass, strings.TrimSpace(match[1]) != "", match[2])
	}
	for _, match := range interfaceRE.FindAllStringSubmatch(source, -1) {
		appendSym(SymbolInterface, strings.TrimSpace(match[1]) != "", match[2])
	}
	return out
}

func parseGoFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range goImportRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}
	for _, match := range goFuncRE.FindAllStringSubmatch(source, -1) {
		out.Symbols = append(out.Symbols, Symbol{Name: match[1], Kind: SymbolFunction, Exported: true})
		out.Exports = append(out.Exports, Export{Name: match[1]})
	}
	for _, match := range goTypeRE.FindAllStringSubmatch(source, -1) {
		out.Symbols = append(out.Symbols, Symbol{Name: match[1], Kind: SymbolStruct, Exported: true})
		out.Exports = append(out.Exports, Export{Name: match[1]})
	}
	return out
}

func parsePythonFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range pyImportRE.FindAllStringSubmatch(source, -1) {
		p := match[1]
		if p == "" {
			p = match[2]
		}
		if p != "" {
			out.Imports = append(out.Imports, Import{ModulePath: p})
		}
	}
	for _, match := range pyDefRE.FindAllStringSubmatch(source, -1) {
		out.Symbols = append(out.Symbols, Symbol{Name: match[1], Kind: SymbolFunction})
	}
	for _, match := range pyClassRE.FindAllStringSubmatch(source, -1) {
		out.Symbols = append(out.Symbols, Symbol{Name: match[1], Kind: SymbolClass})
	}
	return out
}

func parseJavaFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range javaImportRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}
	for _, match := range javaClassRE.FindAllStringSubmatch(source, -1) {
		out.Symbols = append(out.Symbols, Symbol{Name: match[1], Kind: SymbolClass, Exported: true})
		out.Exports = append(out.Exports, Export{Name: match[1]})
	}
	return out
}

func parseCFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range cIncludeRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}
	return out
}

func parsePHPFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range phpUseRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}
	return out
}

func parseCSharpFallback(file ScannedFile, source string) ParsedFile {
	out := ParsedFile{File: file}
	for _, match := range csharpUsingRE.FindAllStringSubmatch(source, -1) {
		out.Imports = append(out.Imports, Import{ModulePath: match[1]})
	}
	for _, match := range csharpClassRE.FindAllStringSubmatch(source, -1) {
		out.Symbols = append(out.Symbols, Symbol{Name: match[1], Kind: SymbolClass, Exported: true})
	}
	return out
}

//go:build !cgo

package indexer

import (
	"fmt"
	"os"
)

// TreeSitterParser uses regex fallbacks when CGO is disabled.
type TreeSitterParser struct{}

func NewTreeSitterParser() *TreeSitterParser { return &TreeSitterParser{} }

func NewTreeSitterTypeScriptParser() *TreeSitterParser { return NewTreeSitterParser() }

func (p *TreeSitterParser) Parse(file ScannedFile) (ParsedFile, error) {
	lang := file.Language
	if lang == "" {
		var ok bool
		lang, ok = LanguageForPath(file.RelativePath)
		if !ok {
			return ParsedFile{File: file}, nil
		}
	}
	sourceBytes, err := os.ReadFile(file.AbsolutePath)
	if err != nil {
		return ParsedFile{}, fmt.Errorf("read source %s: %w", file.RelativePath, err)
	}
	switch lang {
	case LangTypeScript, LangJavaScript:
		return parseTSJSFallback(file, string(sourceBytes)), nil
	case LangGo:
		return parseGoFallback(file, string(sourceBytes)), nil
	case LangPython:
		return parsePythonFallback(file, string(sourceBytes)), nil
	case LangJava:
		return parseJavaFallback(file, string(sourceBytes)), nil
	case LangC, LangCPP:
		return parseCFallback(file, string(sourceBytes)), nil
	case LangPHP:
		return parsePHPFallback(file, string(sourceBytes)), nil
	case LangCSharp:
		return parseCSharpFallback(file, string(sourceBytes)), nil
	default:
		return ParsedFile{File: file}, nil
	}
}

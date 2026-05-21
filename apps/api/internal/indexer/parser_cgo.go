//go:build cgo

package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tscsharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tsc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tscpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tsjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tsjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tsphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tspython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// TreeSitterParser parses supported languages using Tree-sitter grammars.
type TreeSitterParser struct {
	parser *sitter.Parser
}

func NewTreeSitterParser() *TreeSitterParser {
	return &TreeSitterParser{parser: sitter.NewParser()}
}

// NewTreeSitterTypeScriptParser is a backward-compatible alias.
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
	tsLang, err := grammarForLanguage(lang, file.RelativePath)
	if err != nil {
		return ParsedFile{}, err
	}
	p.parser.SetLanguage(tsLang)

	source, err := os.ReadFile(file.AbsolutePath)
	if err != nil {
		return ParsedFile{}, fmt.Errorf("read source %s: %w", file.RelativePath, err)
	}
	tree := p.parser.Parse(source, nil)
	if tree == nil {
		return ParsedFile{}, fmt.Errorf("parse source %s: nil tree", file.RelativePath)
	}
	defer tree.Close()

	out := ParsedFile{File: file}
	walkLanguage(tree.RootNode(), source, lang, false, &out)
	return out, nil
}

func grammarForLanguage(lang Language, relPath string) (*sitter.Language, error) {
	ext := strings.ToLower(filepath.Ext(relPath))
	switch lang {
	case LangTypeScript:
		if ext == ".tsx" {
			return sitter.NewLanguage(tstypescript.LanguageTSX()), nil
		}
		return sitter.NewLanguage(tstypescript.LanguageTypescript()), nil
	case LangJavaScript:
		return sitter.NewLanguage(tsjavascript.Language()), nil
	case LangPython:
		return sitter.NewLanguage(tspython.Language()), nil
	case LangGo:
		return sitter.NewLanguage(tsgo.Language()), nil
	case LangJava:
		return sitter.NewLanguage(tsjava.Language()), nil
	case LangC:
		return sitter.NewLanguage(tsc.Language()), nil
	case LangCPP:
		return sitter.NewLanguage(tscpp.Language()), nil
	case LangPHP:
		return sitter.NewLanguage(tsphp.Language()), nil
	case LangCSharp:
		return sitter.NewLanguage(tscsharp.Language()), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

func walkLanguage(node *sitter.Node, source []byte, lang Language, isExported bool, out *ParsedFile) {
	if node == nil {
		return
	}
	switch lang {
	case LangTypeScript, LangJavaScript:
		walkTSJS(node, source, isExported, out)
	case LangGo:
		walkGo(node, source, isExported, out)
	case LangPython:
		walkPython(node, source, isExported, out)
	case LangJava:
		walkJava(node, source, isExported, out)
	case LangC, LangCPP:
		walkC(node, source, out)
	case LangPHP:
		walkPHP(node, source, isExported, out)
	case LangCSharp:
		walkCSharp(node, source, isExported, out)
	default:
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walkLanguage(node.NamedChild(i), source, lang, isExported, out)
		}
	}
}

func walkTSJS(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "import_statement":
		if imp, ok := extractImport(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "export_statement":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walkTSJS(node.NamedChild(i), source, true, out)
		}
		out.Exports = append(out.Exports, extractExplicitExports(node, source)...)
		return
	case "function_declaration":
		if sym, ok := extractSymbol(node, source, SymbolFunction, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	case "class_declaration":
		if sym, ok := extractSymbol(node, source, SymbolClass, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	case "interface_declaration":
		if sym, ok := extractSymbol(node, source, SymbolInterface, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkTSJS(node.NamedChild(i), source, isExported, out)
	}
}

func walkGo(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "import_declaration":
		if imp, ok := extractGoImport(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "function_declaration", "method_declaration":
		kind := SymbolFunction
		if node.Type() == "method_declaration" {
			kind = SymbolMethod
		}
		if sym, ok := extractSymbol(node, source, kind, goExported(node, source)); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	case "type_declaration":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "type_spec" {
				if sym, ok := extractSymbol(child, source, SymbolStruct, goExported(child, source)); ok {
					out.Symbols = append(out.Symbols, sym)
					if sym.Exported {
						out.Exports = append(out.Exports, Export{Name: sym.Name})
					}
				}
			}
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkGo(node.NamedChild(i), source, isExported, out)
	}
}

func walkPython(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "import_statement", "import_from_statement":
		if imp, ok := extractPythonImport(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "function_definition":
		if sym, ok := extractSymbol(node, source, SymbolFunction, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	case "class_definition":
		if sym, ok := extractSymbol(node, source, SymbolClass, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkPython(node.NamedChild(i), source, isExported, out)
	}
}

func walkJava(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "import_declaration":
		if imp, ok := extractJavaImport(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "class_declaration", "interface_declaration":
		kind := SymbolClass
		if node.Type() == "interface_declaration" {
			kind = SymbolInterface
		}
		if sym, ok := extractSymbol(node, source, kind, true); ok {
			out.Symbols = append(out.Symbols, sym)
			out.Exports = append(out.Exports, Export{Name: sym.Name})
		}
	case "method_declaration":
		if sym, ok := extractSymbol(node, source, SymbolMethod, true); ok {
			out.Symbols = append(out.Symbols, sym)
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkJava(node.NamedChild(i), source, isExported, out)
	}
}

func walkC(node *sitter.Node, source []byte, out *ParsedFile) {
	switch node.Type() {
	case "preproc_include":
		if imp, ok := extractCInclude(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "function_definition":
		if sym, ok := extractSymbol(node, source, SymbolFunction, false); ok {
			out.Symbols = append(out.Symbols, sym)
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkC(node.NamedChild(i), source, out)
	}
}

func walkPHP(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "namespace_use_declaration", "use_declaration":
		if imp, ok := extractPHPUse(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "function_definition":
		if sym, ok := extractSymbol(node, source, SymbolFunction, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
		}
	case "class_declaration":
		if sym, ok := extractSymbol(node, source, SymbolClass, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkPHP(node.NamedChild(i), source, isExported, out)
	}
}

func walkCSharp(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "using_directive":
		if imp, ok := extractCSharpUsing(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "class_declaration":
		if sym, ok := extractSymbol(node, source, SymbolClass, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
			if sym.Exported {
				out.Exports = append(out.Exports, Export{Name: sym.Name})
			}
		}
	case "method_declaration":
		if sym, ok := extractSymbol(node, source, SymbolMethod, isExported); ok {
			out.Symbols = append(out.Symbols, sym)
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkCSharp(node.NamedChild(i), source, isExported, out)
	}
}

func extractGoImport(node *sitter.Node, source []byte) (Import, bool) {
	pathNode := node.ChildByFieldName("path")
	if pathNode != nil {
		return Import{ModulePath: unquote(string(source[pathNode.StartByte():pathNode.EndByte()]))}, true
	}
	return Import{}, false
}

func extractPythonImport(node *sitter.Node, source []byte) (Import, bool) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "dotted_name" || child.Type() == "aliased_import" {
			return Import{ModulePath: string(source[child.StartByte():child.EndByte()])}, true
		}
	}
	return Import{}, false
}

func extractJavaImport(node *sitter.Node, source []byte) (Import, bool) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "scoped_identifier" || child.Type() == "identifier" {
			return Import{ModulePath: string(source[child.StartByte():child.EndByte()])}, true
		}
	}
	return Import{}, false
}

func extractCInclude(node *sitter.Node, source []byte) (Import, bool) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "string_literal" || child.Type() == "system_lib_string" {
			p := unquote(string(source[child.StartByte():child.EndByte()]))
			if strings.HasPrefix(p, ".") {
				return Import{ModulePath: p}, true
			}
		}
	}
	return Import{}, false
}

func extractPHPUse(node *sitter.Node, source []byte) (Import, bool) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "namespace_name" {
			return Import{ModulePath: string(source[child.StartByte():child.EndByte()])}, true
		}
	}
	return Import{}, false
}

func extractCSharpUsing(node *sitter.Node, source []byte) (Import, bool) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "identifier" || child.Type() == "qualified_name" {
			return Import{ModulePath: string(source[child.StartByte():child.EndByte()])}, true
		}
	}
	return Import{}, false
}

func goExported(node *sitter.Node, source []byte) bool {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	name := string(source[nameNode.StartByte():nameNode.EndByte()])
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

func extractImport(node *sitter.Node, source []byte) (Import, bool) {
	var modulePath string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "string" {
			modulePath = unquote(string(source[child.StartByte():child.EndByte()]))
			break
		}
	}
	if modulePath == "" {
		return Import{}, false
	}
	isTypeOnly := strings.Contains(string(source[node.StartByte():node.EndByte()]), "import type")
	return Import{ModulePath: modulePath, TypeOnly: isTypeOnly}, true
}

func extractExplicitExports(node *sitter.Node, source []byte) []Export {
	exports := make([]Export, 0)
	text := string(source[node.StartByte():node.EndByte()])
	if strings.Contains(text, "export default") {
		exports = append(exports, Export{Name: "default"})
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "export_clause" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() == "export_specifier" {
					nameNode := spec.ChildByFieldName("name")
					if nameNode != nil {
						exports = append(exports, Export{Name: string(source[nameNode.StartByte():nameNode.EndByte()])})
					}
				}
			}
		}
	}
	return exports
}

func extractSymbol(node *sitter.Node, source []byte, kind SymbolKind, exported bool) (Symbol, bool) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return Symbol{}, false
	}
	start := nameNode.StartPoint()
	end := nameNode.EndPoint()
	return Symbol{
		Name:      string(source[nameNode.StartByte():nameNode.EndByte()]),
		Kind:      kind,
		Exported:  exported,
		StartLine: int(start.Row) + 1,
		StartCol:  int(start.Column) + 1,
		EndLine:   int(end.Row) + 1,
		EndCol:    int(end.Column) + 1,
	}, true
}

func unquote(v string) string { return strings.Trim(v, "\"'<>") }

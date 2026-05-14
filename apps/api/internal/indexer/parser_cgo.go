//go:build cgo

package indexer

import (
	"fmt"
	"os"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type TreeSitterTypeScriptParser struct {
	parser *sitter.Parser
}

func NewTreeSitterTypeScriptParser() *TreeSitterTypeScriptParser {
	p := sitter.NewParser()
	p.SetLanguage(sitter.NewLanguage(tstypescript.LanguageTypescript()))
	return &TreeSitterTypeScriptParser{parser: p}
}

func (p *TreeSitterTypeScriptParser) Parse(file ScannedFile) (ParsedFile, error) {
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
	walkNode(tree.RootNode(), source, false, &out)
	return out, nil
}

func walkNode(node *sitter.Node, source []byte, isExported bool, out *ParsedFile) {
	switch node.Type() {
	case "import_statement":
		if imp, ok := extractImport(node, source); ok {
			out.Imports = append(out.Imports, imp)
		}
	case "export_statement":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walkNode(node.NamedChild(i), source, true, out)
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
		walkNode(node.NamedChild(i), source, isExported, out)
	}
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
		if child.Type() == "string" {
			src := unquote(string(source[child.StartByte():child.EndByte()]))
			if len(exports) == 0 {
				exports = append(exports, Export{SourcePath: src})
			}
			for idx := range exports {
				exports[idx].SourcePath = src
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

func unquote(v string) string { return strings.Trim(v, "\"'") }

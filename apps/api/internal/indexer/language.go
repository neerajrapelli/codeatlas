package indexer

import (
	"path/filepath"
	"strings"
)

// Language identifies which Tree-sitter grammar / fallback parser to use.
type Language string

const (
	LangTypeScript Language = "typescript"
	LangJavaScript Language = "javascript"
	LangPython     Language = "python"
	LangGo         Language = "go"
	LangJava       Language = "java"
	LangC          Language = "c"
	LangCPP        Language = "cpp"
	LangPHP        Language = "php"
	LangCSharp     Language = "csharp"
)

type extRule struct {
	ext  string
	lang Language
}

var extensionRules = []extRule{
	{".ts", LangTypeScript}, {".mts", LangTypeScript}, {".cts", LangTypeScript},
	{".tsx", LangTypeScript},
	{".js", LangJavaScript}, {".mjs", LangJavaScript}, {".cjs", LangJavaScript},
	{".jsx", LangJavaScript},
	{".py", LangPython},
	{".go", LangGo},
	{".java", LangJava},
	{".c", LangC}, {".h", LangC},
	{".cpp", LangCPP}, {".cc", LangCPP}, {".cxx", LangCPP},
	{".hpp", LangCPP}, {".hh", LangCPP}, {".hxx", LangCPP},
	{".php", LangPHP},
	{".cs", LangCSharp},
}

// LanguageForPath returns the language for a repository-relative path.
func LanguageForPath(relPath string) (Language, bool) {
	ext := strings.ToLower(filepath.Ext(relPath))
	for _, rule := range extensionRules {
		if rule.ext == ext {
			return rule.lang, true
		}
	}
	return "", false
}

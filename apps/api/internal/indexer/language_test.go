package indexer

import "testing"

func TestLanguageForPath(t *testing.T) {
	cases := []struct {
		path string
		want Language
	}{
		{"src/a.ts", LangTypeScript},
		{"src/a.tsx", LangTypeScript},
		{"lib/util.js", LangJavaScript},
		{"main.go", LangGo},
		{"app.py", LangPython},
		{"Main.java", LangJava},
		{"hdr.h", LangC},
		{"lib.cpp", LangCPP},
		{"index.php", LangPHP},
		{"Program.cs", LangCSharp},
	}
	for _, tc := range cases {
		got, ok := LanguageForPath(tc.path)
		if !ok || got != tc.want {
			t.Fatalf("%s: got %v %v want %v", tc.path, got, ok, tc.want)
		}
	}
}

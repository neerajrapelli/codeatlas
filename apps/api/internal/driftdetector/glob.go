package driftdetector

import (
	"path/filepath"
	"strings"
)

// matchGlob matches file paths against patterns supporting ** (recursive).
func matchGlob(pattern, path string) bool {
	pattern = normPath(pattern)
	path = normPath(path)
	if pattern == "" {
		return false
	}
	if pattern == "**" || pattern == "*" {
		return true
	}
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		prefix := strings.TrimSuffix(parts[0], "/")
		if len(parts) == 2 && prefix == "" {
			suffix := strings.TrimPrefix(parts[1], "/")
			if strings.HasPrefix(suffix, "*.") {
				ext := strings.TrimPrefix(suffix, "*")
				return strings.HasSuffix(path, ext)
			}
		}
		if prefix != "" && !strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")) {
			if !strings.HasPrefix(path, prefix) {
				return false
			}
		}
		if len(parts) == 1 {
			return strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")) || path == strings.TrimSuffix(prefix, "/")
		}
		suffix := strings.TrimPrefix(parts[1], "/")
		if suffix == "" || suffix == "*" {
			if prefix == "" {
				return true
			}
			p := strings.TrimSuffix(prefix, "/")
			return path == p || strings.HasPrefix(path, p+"/")
		}
		// prefix/**/suffix
		if prefix != "" {
			p := strings.TrimSuffix(prefix, "/")
			if !(path == p || strings.HasPrefix(path, p+"/")) {
				return false
			}
			rest := strings.TrimPrefix(path, p)
			rest = strings.TrimPrefix(rest, "/")
			return strings.Contains("/"+rest+"/", "/"+suffix+"/") ||
				strings.HasSuffix(rest, "/"+suffix) ||
				rest == suffix ||
				strings.HasSuffix(path, suffix)
		}
		return strings.Contains(path, suffix)
	}
	ok, err := filepath.Match(pattern, path)
	if err != nil {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "/*")) ||
			strings.HasPrefix(path, strings.TrimSuffix(pattern, "\\*"))
	}
	return ok
}

func normPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return p
}

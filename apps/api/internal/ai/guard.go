package ai

import (
	"regexp"
	"strings"
)

var (
	pathLikeRe = regexp.MustCompile(`(?:^|[\s"'(])([\w./-]+\.(?:go|ts|tsx|js|jsx|py|java|cs|cpp|c|h|php|rb|rs|kt|swift|mjs|cjs))(?:[\s"'),.:;]|$)`)
	ruleLikeRe = regexp.MustCompile(`rule\s+["']([^"']+)["']`)
)

// ExtractPathMentions finds file-path-like tokens in assistant text.
func ExtractPathMentions(text string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, m := range pathLikeRe.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		p := strings.TrimPrefix(m[1], "./")
		if p == "" || strings.Contains(p, "..") {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// ExtractRuleMentions finds architecture rule name references in assistant text.
func ExtractRuleMentions(text string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, m := range ruleLikeRe.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		n := strings.TrimSpace(m[1])
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// SanitizeAnswer marks unknown paths/rules in streamed answer text for UI rendering.
func SanitizeAnswer(text string, pathOK, ruleOK map[string]bool) string {
	if text == "" {
		return text
	}
	out := text
	for path, ok := range pathOK {
		if ok {
			continue
		}
		out = strings.ReplaceAll(out, path, "⟨unverified:"+path+"⟩")
	}
	for rule, ok := range ruleOK {
		if ok {
			continue
		}
		out = strings.ReplaceAll(out, `rule "`+rule+`"`, `rule "⟨unverified:`+rule+`⟩"`)
	}
	return out
}

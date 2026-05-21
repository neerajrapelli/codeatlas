package signals

import (
	"regexp"
	"strings"
)

// Allowed signal_type values (architecture_signals CHECK constraint).
const (
	TypeTechnicalDebt         = "technical_debt"
	TypeCouplingWarning       = "coupling_warning"
	TypeMigrationIntent       = "migration_intent"
	TypeKnownFragility        = "known_fragility"
	TypeOwnershipBoundary     = "ownership_boundary"
	TypeArchitecturalDecision = "architectural_decision"
)

// Draft is a candidate architecture signal before persistence.
type Draft struct {
	SignalType string
	Summary    string
	Confidence float64
	FilePaths  []string
}

var pathPattern = regexp.MustCompile(
	`(?:^|[\s"'(\[])((?:[\w.-]+/)+[\w.-]+\.(?:ts|tsx|js|jsx|mjs|cjs|go|py|rs|java|kt|cs|php|rb|md|json|ya?ml|sql|vue|svelte|swift|scala))`,
)

type rule struct {
	signalType string
	keywords   []string
	confidence float64
}

var rules = []rule{
	{TypeTechnicalDebt, []string{"tech debt", "technical debt", "todo:", "fixme", "hack", "workaround", "needs refactor", "refactor later"}, 0.78},
	{TypeCouplingWarning, []string{"coupling", "tightly coupled", "circular dependency", "god object", "spaghetti"}, 0.8},
	{TypeMigrationIntent, []string{"migrate to", "migration plan", "deprecate", "sunset", "replace with"}, 0.82},
	{TypeKnownFragility, []string{"fragile", "flaky", "brittle", "race condition", "known issue", "breaks often"}, 0.85},
	{TypeOwnershipBoundary, []string{"ownership", "team boundary", "on-call", "code owner", "bus factor"}, 0.75},
	{TypeArchitecturalDecision, []string{"adr", "architecture decision", "we decided", "design decision", "rfc"}, 0.88},
}

// ExtractFromText scans PR/issue/comment bodies for normalized architecture signals.
func ExtractFromText(text string) []Draft {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	lower := strings.ToLower(text)
	paths := extractPaths(text)
	var out []Draft
	seen := make(map[string]struct{})
	for _, r := range rules {
		if !containsAny(lower, r.keywords) {
			continue
		}
		key := r.signalType
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Draft{
			SignalType: r.signalType,
			Summary:    summarize(text, 220),
			Confidence: r.confidence,
			FilePaths:  paths,
		})
	}
	return out
}

func extractPaths(text string) []string {
	matches := pathPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]struct{})
	var paths []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		p := normalizePath(m[1])
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	return paths
}

func normalizePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return p
}

func containsAny(lower string, keywords []string) bool {
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func summarize(text string, max int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= max {
		return text
	}
	return text[:max-1] + "…"
}

package archintel

import (
	"strings"

	"codeatlas/apps/api/internal/signals"
)

type DiscussionInput struct {
	SourceKind string
	SourceRef  string
	Title      string
	Body       string
	Author     string
	OccurredAt string
}

type Analyzer struct {
	enableLLM bool
}

func NewAnalyzer(enableLLM bool) *Analyzer {
	return &Analyzer{enableLLM: enableLLM}
}

// AnalyzeDiscussion extracts heuristic architecture decision candidates.
// LLM enrichment is optional and can be wired later without breaking contracts.
func (a *Analyzer) AnalyzeDiscussion(in DiscussionInput) []DecisionRecord {
	text := strings.TrimSpace(in.Title + "\n" + in.Body)
	if text == "" {
		return nil
	}
	drafts := signals.ExtractFromText(text)
	out := make([]DecisionRecord, 0, len(drafts))
	for _, d := range drafts {
		status := DecisionProposed
		lower := strings.ToLower(text)
		switch {
		case strings.Contains(lower, "accepted"), strings.Contains(lower, "approved"), strings.Contains(lower, "merge"):
			status = DecisionAccepted
		case strings.Contains(lower, "rejected"), strings.Contains(lower, "declined"), strings.Contains(lower, "won't do"):
			status = DecisionRejected
		}
		participants := []string{}
		if strings.TrimSpace(in.Author) != "" {
			participants = append(participants, in.Author)
		}
		rec := DecisionRecord{
			Title:           truncateTitle(d.Summary),
			Summary:         d.Summary,
			Status:          status,
			Confidence:      d.Confidence,
			Tradeoffs:       inferTradeoffs(text),
			AffectedModules: uniqueStrings(d.FilePaths),
			AffectedFiles:   uniqueStrings(d.FilePaths),
			Participants:    participants,
		}
		out = append(out, rec)
	}
	_ = a.enableLLM // placeholder for optional enrichment wiring.
	return out
}

func inferTradeoffs(text string) []string {
	text = strings.ToLower(text)
	tradeoffs := make([]string, 0, 4)
	candidates := []string{
		"performance vs complexity",
		"latency vs consistency",
		"throughput vs readability",
		"cost vs reliability",
		"developer velocity vs strictness",
	}
	for _, c := range candidates {
		parts := strings.Split(c, " vs ")
		if len(parts) != 2 {
			continue
		}
		if strings.Contains(text, parts[0]) && strings.Contains(text, parts[1]) {
			tradeoffs = append(tradeoffs, c)
		}
	}
	return tradeoffs
}

func truncateTitle(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 96 {
		return s
	}
	return s[:96]
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

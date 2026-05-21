package ai

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are CodeAtlas architecture assistant.
Answer precisely using provided repository context only.
Explain impact and dependency paths.
If context is insufficient, say what is missing.
Always include a "Related files" section with bullet paths.`

func buildPrompt(query string, contextItems []ContextItem, maxChars int) []string {
	parts := []string{
		fmt.Sprintf("Question: %s", query),
		"Context:",
	}
	used := len(parts[0]) + len(parts[1])

	for _, item := range contextItems {
		socio := ""
		if item.DominantOwnerLogin != "" || item.RiskLevel != "" || item.IsHotspot || item.HasBusFactorRisk {
			socio = fmt.Sprintf(
				" owner=%s bus_factor=%d churn_90d=%.0f risk=%s hotspot=%v bus_risk=%v commits_90d=%d",
				item.DominantOwnerLogin, item.BusFactor, item.ChurnScore, item.RiskLevel,
				item.IsHotspot, item.HasBusFactorRisk, item.CommitCount90d,
			)
		}
		block := fmt.Sprintf(
			"- file=%s dep_out=%d dep_in=%d imports=%v exports=%v symbols=%v%s",
			item.Path, item.DependencyOut, item.DependencyIn, item.Imports, item.Exports, item.Symbols, socio,
		)
		if used+len(block) > maxChars {
			break
		}
		used += len(block)
		parts = append(parts, block)
	}
	return parts
}

func buildUserPrompt(query string, contextItems []ContextItem, maxChars int) string {
	return strings.Join(buildPrompt(query, contextItems, maxChars), "\n")
}

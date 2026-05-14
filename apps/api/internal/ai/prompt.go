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
		block := fmt.Sprintf(
			"- file=%s dep_out=%d dep_in=%d imports=%v exports=%v symbols=%v",
			item.Path, item.DependencyOut, item.DependencyIn, item.Imports, item.Exports, item.Symbols,
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
